"""OpenClaw Gateway client using the OpenAI-compatible chat completions API."""

from __future__ import annotations

import json
import urllib.error
import urllib.request
from dataclasses import dataclass
from http.client import HTTPResponse
from typing import cast

from ..core.config import settings

type ObjectDict = dict[str, object]


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

    def ask(self, question: str, scope: ObjectDict, context: ObjectDict) -> ObjectDict:
        prompt = [
            "请基于给定业务上下文回答问题。",
            "如果上下文不足，必须明确指出缺少哪些信息，不要编造未提供的事实。",
            f"问题：{question}",
            "范围：",
            json.dumps(scope, ensure_ascii=False, indent=2),
            "业务上下文：",
            json.dumps(context, ensure_ascii=False, indent=2),
        ]
        result = self._chat(
            system_prompt=(
                "你是 DigiDocs 科研文档资产管理平台的 AI 助手。"
                "你的回答必须忠于提供的上下文，只输出中文。"
            ),
            user_prompt="\n".join(prompt),
        )
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
        raw = self._structured_chat(
            system_prompt=(
                "你是 DigiDocs 文档助手。"
                "请根据提供的业务上下文生成结构化中文摘要。"
                "如果缺少文档正文，必须明确声明这是基于元数据和历史记录的摘要。"
            ),
            user_prompt=(
                "任务类型：document.summarize\n"
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
            raw.get("suggestions"), default_type="document_summary"
        )
        return {
            "task_type": "document.summarize",
            "summary_text": _string_value(raw.get("summary_text")),
            "suggestions": suggestions,
        }

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
        result = self._chat(system_prompt=system_prompt, user_prompt=user_prompt)
        content = result.content.strip()
        try:
            return _parse_json_content(content)
        except json.JSONDecodeError as exc:
            raise OpenClawClientError(f"OpenClaw 返回了非 JSON 结构化结果: {exc}") from exc

    def _chat(self, system_prompt: str, user_prompt: str) -> ChatCompletionResult:
        payload: ObjectDict = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
        }
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

    def _post(self, path: str, payload: ObjectDict) -> ObjectDict:
        url = f"{self.base_url}{path}"
        request = urllib.request.Request(
            url,
            data=json.dumps(payload).encode(),
            method="POST",
        )
        request.add_header("Content-Type", "application/json")
        request.add_header("Accept", "application/json")
        if self.api_key and self.api_key != "replace-me":
            request.add_header("Authorization", f"Bearer {self.api_key}")
        if self.backend_model:
            request.add_header("x-openclaw-model", self.backend_model)

        try:
            with cast(
                HTTPResponse, urllib.request.urlopen(request, timeout=self.timeout_seconds)
            ) as response:
                body = response.read().decode()
        except urllib.error.HTTPError as exc:
            error_body = exc.read().decode(errors="ignore")
            raise OpenClawClientError(
                f"OpenClaw HTTP {exc.code}: {error_body or exc.reason}"
            ) from exc
        except Exception as exc:
            raise OpenClawClientError(f"OpenClaw 请求失败: {exc}") from exc

        try:
            parsed = cast(object, json.loads(body))
        except json.JSONDecodeError as exc:
            raise OpenClawClientError("OpenClaw 返回了非 JSON 响应") from exc
        parsed_dict = _as_object_dict(parsed)
        if parsed_dict is None:
            raise OpenClawClientError("OpenClaw 返回结构不是对象")
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


def _parse_json_content(content: str) -> ObjectDict:
    cleaned = content.strip()
    if cleaned.startswith("```"):
        cleaned = cleaned.strip("`")
        cleaned = cleaned.removeprefix("json").strip()
    parsed = cast(object, json.loads(cleaned))
    parsed_dict = _as_object_dict(parsed)
    if parsed_dict is None:
        raise json.JSONDecodeError("top-level JSON must be object", cleaned, 0)
    return parsed_dict


def _normalize_suggestions(raw: object, default_type: str) -> list[ObjectDict]:
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
        normalized.append(suggestion)

    return normalized


def _string_value(value: object) -> str:
    return value if isinstance(value, str) else ""


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None
