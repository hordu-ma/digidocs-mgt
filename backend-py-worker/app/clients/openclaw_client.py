"""OpenClaw Gateway client using the OpenAI-compatible chat completions API."""

from __future__ import annotations

import json
import logging
import time
import urllib.request
from collections.abc import Callable
from dataclasses import dataclass
from typing import cast

from ..services.context_budget import split_for_summary

from ..core.config import settings
from .http_util import HttpError, fetch

type ObjectDict = dict[str, object]

logger = logging.getLogger(__name__)


class OpenClawClientError(RuntimeError):
    """Raised when the OpenClaw gateway returns an unusable response."""


@dataclass(slots=True)
class ChatCompletionResult:
    response_id: str
    model: str
    content: str
    usage: ObjectDict


class OpenClawClient:
    def __init__(self) -> None:
        self.base_url: str = settings.openclaw_base_url.rstrip("/")
        self.api_key: str = settings.openclaw_api_key
        self.model: str = settings.openclaw_model
        self.backend_model: str = settings.openclaw_backend_model
        self.timeout_seconds: int = settings.openclaw_timeout_seconds
        # Flipped off the first time the gateway rejects response_format, so
        # later calls do not pay for a guaranteed 400.
        self._json_mode: bool = True

    def ask(
        self,
        question: str,
        scope: ObjectDict,
        context: ObjectDict,
        on_delta: Callable[[str], None] | None = None,
    ) -> ObjectDict:
        payload = _as_object_dict(context.get("payload")) or {}
        memory = _as_object_dict(payload.get("memory")) or {}
        prompt = [
            "请基于给定业务上下文回答问题。",
            "如果上下文不足，必须明确指出缺少哪些信息，不要编造未提供的事实。",
            "业务上下文中的 passages 是从文档正文检索出的片段；引用其中内容时，在句末用 [片段N] 标注来源。",
            f"问题：{question}",
            "范围：",
            json.dumps(scope, ensure_ascii=False, indent=2),
            "显式记忆：",
            json.dumps(memory, ensure_ascii=False, indent=2),
            "业务上下文：",
            json.dumps(context, ensure_ascii=False, indent=2),
        ]
        system_prompt = (
            "你是 DigiDocs 科研文档资产管理平台的 AI 助手。"
            "你的回答必须忠于提供的上下文，只输出中文。"
        )
        user_prompt = "\n".join(prompt)
        if on_delta is not None and settings.stream_answers:
            result = self._chat_stream(system_prompt, user_prompt, on_delta)
        else:
            result = self._chat(system_prompt=system_prompt, user_prompt=user_prompt)
        return {
            "request_id": result.response_id,
            "answer": result.content,
            "model": result.model,
            "usage": result.usage,
            "source_scope": scope,
        }

    def summarize_document(
        self,
        request_id: str,
        payload: ObjectDict,
        context: ObjectDict,
    ) -> ObjectDict:
        context, basis = self._fit_document_text(context)
        raw = self._structured_chat(
            system_prompt=(
                "你是 DigiDocs 文档助手。"
                "请根据提供的业务上下文生成结构化中文摘要。"
                "如果缺少文档正文，必须明确声明这是基于元数据和历史记录的摘要。"
                "如果正文是分段要点汇总，或注明为抽样，请在摘要中如实说明。"
            ),
            user_prompt=(
                "任务类型：document.summarize\n"
                f"request_id: {request_id}\n"
                "请输出 JSON，字段固定为："
                "`summary_text`, `suggestions`。"
                "`suggestions` 为数组，每项包含 `title`, `content`, `suggestion_type`, `confidence`，"
                "以及可选的 `suggested_action`：仅当正文和流转记录明确表明应当如此时，"
                "才可取 `mark_in_progress`、`finalize`、`archive` 之一，否则省略。"
                "该动作只作为建议，需人工确认后才会执行。\n"
                "任务负载：\n"
                f"{json.dumps(payload, ensure_ascii=False, indent=2)}\n"
                "业务上下文：\n"
                f"{json.dumps(context, ensure_ascii=False, indent=2)}"
            ),
        )
        suggestions = _normalize_suggestions(
            raw.get("suggestions"), default_type="document_summary", allow_action=True
        )
        return {
            "task_type": "document.summarize",
            "summary_text": _string_value(raw.get("summary_text")),
            "suggestions": suggestions,
            "summary_basis": basis,
        }

    def _fit_document_text(self, context: ObjectDict) -> tuple[ObjectDict, str]:
        """Shrink context["document_text"] to the prompt budget.

        Returns the (copied) context and the basis of the summary:
        "full_text", "map_reduce", "map_reduce_sampled" or "metadata_only".
        Long texts are summarized section by section first (map), and the final
        structured summary is written from those section notes (reduce).
        """
        context = dict(context)
        text = _string_value(context.get("document_text"))
        doc_ctx = _as_object_dict(context.get("document_context"))
        if doc_ctx is not None and "extracted_text" in doc_ctx:
            # Same text as document_text; sending it twice doubles the prompt.
            doc_ctx = dict(doc_ctx)
            doc_ctx.pop("extracted_text", None)
            context["document_context"] = doc_ctx
        if not text:
            return context, "metadata_only"
        budget = settings.max_context_chars
        if len(text) <= budget:
            return context, "full_text"
        sections, sampled = split_for_summary(
            text, budget, max(1, settings.max_summary_sections)
        )
        notes: list[str] = []
        for index, section in enumerate(sections, start=1):
            result = self._chat(
                system_prompt="你是科研文档摘要助手，只输出中文要点，不要编造。",
                user_prompt=f"以下是一份长文档的第 {index}/{len(sections)} 部分。请用不超过 200 字列出本部分要点。\n\n{section}",
            )
            notes.append(f"【第 {index} 部分要点】{result.content}")
        header = f"（原文约 {len(text)} 字，分 {len(sections)} 段提炼"
        header += "，因篇幅过长为均匀抽样，未覆盖全部段落）" if sampled else "）"
        context["document_text"] = header + "\n" + "\n".join(notes)
        return context, "map_reduce_sampled" if sampled else "map_reduce"

    def summarize_handover(
        self,
        request_id: str,
        payload: ObjectDict,
        context: ObjectDict,
    ) -> ObjectDict:
        raw = self._structured_chat(
            system_prompt=(
                "你是 DigiDocs 交接助手。"
                "请根据提供的交接信息生成结构化中文摘要，突出待确认事项与潜在风险。"
            ),
            user_prompt=(
                "任务类型：handover.summarize\n"
                f"request_id: {request_id}\n"
                "请输出 JSON，字段固定为："
                "`summary_text`, `suggestions`。"
                "`suggestions` 为数组，每项包含 `title`, `content`, `suggestion_type`, `confidence`。\n"
                "任务负载：\n"
                f"{json.dumps(payload, ensure_ascii=False, indent=2)}\n"
                "业务上下文：\n"
                f"{json.dumps(context, ensure_ascii=False, indent=2)}"
            ),
        )
        suggestions = _normalize_suggestions(
            raw.get("suggestions"), default_type="handover_summary"
        )
        return {
            "task_type": "handover.summarize",
            "summary_text": _string_value(raw.get("summary_text")),
            "suggestions": suggestions,
        }

    def generate_suggestion(
        self,
        request_id: str,
        payload: ObjectDict,
        context: ObjectDict,
    ) -> ObjectDict:
        raw = self._structured_chat(
            system_prompt=(
                "你是 DigiDocs 结构化建议助手。"
                "请根据业务上下文产出可执行建议，不要输出业务主状态修改指令。"
            ),
            user_prompt=(
                "任务类型：assistant.generate_suggestion\n"
                f"request_id: {request_id}\n"
                "请输出 JSON，字段固定为：`suggestions`。"
                "`suggestions` 为数组，每项包含 `title`, `content`, `suggestion_type`, `confidence`。\n"
                "任务负载：\n"
                f"{json.dumps(payload, ensure_ascii=False, indent=2)}\n"
                "业务上下文：\n"
                f"{json.dumps(context, ensure_ascii=False, indent=2)}"
            ),
        )
        return {
            "task_type": "assistant.generate_suggestion",
            "suggestions": _normalize_suggestions(
                raw.get("suggestions"),
                default_type="structure_recommendation",
            ),
        }

    def _structured_chat(self, system_prompt: str, user_prompt: str) -> ObjectDict:
        """Ask for a JSON object.

        Uses response_format={"type": "json_object"} when the gateway accepts
        it (probed 2026-11-10: json_object works, json_schema is rejected by
        the backend models). If the reply still is not valid JSON, one repair
        round asks the model to re-emit only the object.
        """
        result = self._chat_json_mode(system_prompt, user_prompt)
        content = result.content.strip()
        try:
            return _parse_json_content(content)
        except json.JSONDecodeError:
            logger.warning("structured output not JSON, attempting one repair round")
        repaired = self._chat(
            system_prompt="你只输出一个合法的 JSON 对象，不要输出任何解释、注释或 Markdown 代码块。",
            user_prompt="把下面的内容整理成同样字段的合法 JSON 对象：\n" + content[:6000],
        )
        try:
            return _parse_json_content(repaired.content)
        except json.JSONDecodeError as exc:
            raise OpenClawClientError(f"OpenClaw 返回了非 JSON 结构化结果: {exc}") from exc

    def _chat_json_mode(self, system_prompt: str, user_prompt: str) -> ChatCompletionResult:
        if self._json_mode:
            try:
                return self._chat(
                    system_prompt=system_prompt,
                    user_prompt=user_prompt,
                    response_format={"type": "json_object"},
                )
            except OpenClawClientError as exc:
                if "HTTP 400" not in str(exc):
                    raise
                logger.warning("gateway rejected response_format json_object; falling back to prompt-only JSON")
                self._json_mode = False
        return self._chat(system_prompt=system_prompt, user_prompt=user_prompt)

    def _chat_stream(
        self,
        system_prompt: str,
        user_prompt: str,
        on_delta: Callable[[str], None],
    ) -> ChatCompletionResult:
        """Stream a completion, calling on_delta with the text so far.

        Falls back to a normal call if the stream fails before any token
        arrived; a failure mid-stream is an error (the partial answer must not
        be reported as complete).
        """
        payload: ObjectDict = {
            "model": self.model,
            "stream": True,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
        }
        request = self._build_request("/v1/chat/completions", payload, accept="text/event-stream")
        started_at = time.perf_counter()
        parts: list[str] = []
        response_id = ""
        model = ""
        usage: ObjectDict = {}
        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as resp:
                for raw_line in resp:
                    line = raw_line.decode("utf-8", errors="ignore").strip()
                    if not line.startswith("data:"):
                        continue
                    data = line[5:].strip()
                    if data == "[DONE]":
                        break
                    try:
                        event = _as_object_dict(cast(object, json.loads(data))) or {}
                    except json.JSONDecodeError:
                        continue
                    response_id = response_id or _string_value(event.get("id"))
                    model = model or _string_value(event.get("model"))
                    usage = _as_object_dict(event.get("usage")) or usage
                    delta = _extract_delta(event)
                    if delta:
                        parts.append(delta)
                        on_delta("".join(parts))
        except Exception as exc:
            if not parts:
                logger.warning("openclaw stream failed before first token, retrying without stream: %s", exc)
                return self._chat(system_prompt=system_prompt, user_prompt=user_prompt)
            raise OpenClawClientError(f"OpenClaw 流式输出中断: {exc}") from exc
        content = "".join(parts).strip()
        if not content:
            raise OpenClawClientError("OpenClaw 返回内容为空")
        logger.info(
            "openclaw stream ok duration_ms=%d chars=%d",
            int((time.perf_counter() - started_at) * 1000),
            len(content),
        )
        return ChatCompletionResult(
            response_id=response_id or "openclaw-response",
            model=model or self.model,
            content=content,
            usage=usage,
        )

    def _chat(
        self,
        system_prompt: str,
        user_prompt: str,
        response_format: ObjectDict | None = None,
    ) -> ChatCompletionResult:
        payload: ObjectDict = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
        }
        if response_format is not None:
            payload["response_format"] = response_format
        raw = self._post("/v1/chat/completions", payload)
        content = _extract_message_content(raw)
        if content == "":
            raise OpenClawClientError("OpenClaw 返回内容为空")
        return ChatCompletionResult(
            response_id=_string_value(raw.get("id")) or "openclaw-response",
            model=_string_value(raw.get("model")) or self.model,
            content=content,
            usage=_as_object_dict(raw.get("usage")) or {},
        )

    def _build_request(self, path: str, payload: ObjectDict, accept: str = "application/json") -> urllib.request.Request:
        request = urllib.request.Request(
            f"{self.base_url}{path}",
            data=json.dumps(payload).encode(),
            method="POST",
        )
        request.add_header("Content-Type", "application/json")
        request.add_header("Accept", accept)
        if self.api_key and self.api_key != "replace-me":
            request.add_header("Authorization", f"Bearer {self.api_key}")
        if self.backend_model:
            request.add_header("x-openclaw-model", self.backend_model)
        return request

    def _post(self, path: str, payload: ObjectDict) -> ObjectDict:
        request = self._build_request(path, payload)

        started_at = time.perf_counter()
        logger.info("calling openclaw path=%s model=%s", path, payload.get("model", self.model))
        try:
            _, _, raw = fetch(request, timeout=self.timeout_seconds, label=f"openclaw {path}")
            body = raw.decode()
        except HttpError as exc:
            error_body = exc.body.decode(errors="ignore")
            logger.warning(
                "openclaw http error path=%s status=%s duration_ms=%d",
                path,
                exc.status,
                int((time.perf_counter() - started_at) * 1000),
            )
            raise OpenClawClientError(
                f"OpenClaw HTTP {exc.status}: {error_body or exc.reason}"
            ) from exc
        except Exception as exc:
            logger.warning(
                "openclaw request failed path=%s duration_ms=%d error=%s",
                path,
                int((time.perf_counter() - started_at) * 1000),
                exc,
            )
            raise OpenClawClientError(f"OpenClaw 请求失败: {exc}") from exc

        try:
            parsed = cast(object, json.loads(body))
        except json.JSONDecodeError as exc:
            raise OpenClawClientError("OpenClaw 返回了非 JSON 响应") from exc
        parsed_dict = _as_object_dict(parsed)
        if parsed_dict is None:
            raise OpenClawClientError("OpenClaw 返回结构不是对象")
        logger.info(
            "openclaw response ok path=%s duration_ms=%d response_id=%s model=%s",
            path,
            int((time.perf_counter() - started_at) * 1000),
            parsed_dict.get("id", ""),
            parsed_dict.get("model", self.model),
        )
        return parsed_dict


def _extract_message_content(payload: ObjectDict) -> str:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices:
        return ""
    choice_items = cast(list[object], choices)
    first_choice = _as_object_dict(choice_items[0])
    if first_choice is None:
        return ""
    message = _as_object_dict(first_choice.get("message"))
    if message is None:
        return ""
    content = message.get("content")
    if isinstance(content, str):
        return content.strip()
    if isinstance(content, list):
        parts: list[str] = []
        for item in cast(list[object], content):
            item_dict = _as_object_dict(item)
            if item_dict is not None and item_dict.get("type") == "text":
                text = item_dict.get("text")
                if isinstance(text, str) and text.strip():
                    parts.append(text.strip())
        return "\n".join(parts).strip()
    return ""


def _extract_delta(event: ObjectDict) -> str:
    choices = event.get("choices")
    if not isinstance(choices, list) or not choices:
        return ""
    first = _as_object_dict(cast(list[object], choices)[0]) or {}
    delta = _as_object_dict(first.get("delta")) or {}
    content = delta.get("content")
    return content if isinstance(content, str) else ""


def _parse_json_content(content: str) -> ObjectDict:
    cleaned = content.strip()
    if cleaned.startswith("```"):
        cleaned = cleaned.strip("`")
        cleaned = cleaned.removeprefix("json").strip()
    if not cleaned.startswith("{"):
        # Models sometimes wrap the object in prose; take the outermost braces.
        start, end = cleaned.find("{"), cleaned.rfind("}")
        if start >= 0 and end > start:
            cleaned = cleaned[start : end + 1]
    parsed = cast(object, json.loads(cleaned))
    parsed_dict = _as_object_dict(parsed)
    if parsed_dict is None:
        raise json.JSONDecodeError("top-level JSON must be object", cleaned, 0)
    return parsed_dict


SUGGESTED_ACTIONS = {"mark_in_progress", "finalize", "archive"}


def _normalize_suggestions(raw: object, default_type: str, allow_action: bool = False) -> list[ObjectDict]:
    if not isinstance(raw, list):
        return []

    normalized: list[ObjectDict] = []
    for item in cast(list[object], raw):
        item_dict = _as_object_dict(item)
        if item_dict is None:
            continue
        content = _string_value(item_dict.get("content"))
        if not content:
            continue
        suggestion: ObjectDict = {
            "title": _string_value(item_dict.get("title")) or "AI 建议",
            "content": content,
            "suggestion_type": _string_value(item_dict.get("suggestion_type")) or default_type,
        }
        confidence = item_dict.get("confidence")
        if isinstance(confidence, (int, float)):
            suggestion["confidence"] = float(confidence)
        action = _string_value(item_dict.get("suggested_action")).strip().lower()
        if allow_action and action in SUGGESTED_ACTIONS:
            suggestion["suggested_action"] = action
        normalized.append(suggestion)

    return normalized


def _string_value(value: object) -> str:
    return value if isinstance(value, str) else ""


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None
