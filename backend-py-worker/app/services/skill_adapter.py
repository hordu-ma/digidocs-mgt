from __future__ import annotations

import json
from typing import cast

from ..clients.openclaw_client import OpenClawClient
from ..tasks.contracts import WorkerTask
from .skill_registry import SkillRegistry, SkillResolutionError

type ObjectDict = dict[str, object]


class SkillAdapterError(RuntimeError):
    """Raised when skill routing or output normalization fails."""


class WorkerSkillAdapter:
    def __init__(
        self,
        openclaw_client: OpenClawClient,
        registry: SkillRegistry | None = None,
    ) -> None:
        self.openclaw_client = openclaw_client
        self.registry = registry or SkillRegistry()

    def run(
        self,
        task: WorkerTask,
        context: ObjectDict,
        scope: ObjectDict | None = None,
    ) -> ObjectDict:
        normalized_scope = self._normalize_scope(task, context, scope)
        self._validate_scope(task, normalized_scope)

        try:
            definition = self.registry.resolve(
                task.task_type,
                _string_value(task.payload.get("skill_name")),
            )
        except SkillResolutionError as exc:
            raise SkillAdapterError(str(exc)) from exc

        if definition.name == "answer_with_context":
            raw_output = self.openclaw_client.ask(
                question=_string_value(task.payload.get("question")),
                scope=normalized_scope,
                context=context,
            )
        elif definition.name == "document_summary":
            raw_output = self.openclaw_client.summarize_document(
                task.request_id,
                task.payload,
                context,
            )
        elif definition.name == "handover_summary":
            raw_output = self.openclaw_client.summarize_handover(
                task.request_id,
                task.payload,
                context,
            )
        elif definition.name == "structured_suggestion":
            raw_output = self.openclaw_client.generate_suggestion(
                task.request_id,
                task.payload,
                context,
            )
        else:
            raise SkillAdapterError(f"未实现的 skill: {definition.name}")

        return self._normalize_output(
            task,
            raw_output,
            skill_name=definition.name,
            skill_version=definition.version,
            source_scope=normalized_scope,
        )

    def _normalize_scope(
        self,
        task: WorkerTask,
        context: ObjectDict,
        explicit_scope: ObjectDict | None,
    ) -> ObjectDict:
        result: ObjectDict = {}
        for candidate in (
            _extract_scope_from_payload(task.payload),
            explicit_scope or {},
            _extract_scope_from_context(context),
        ):
            for key in ("project_id", "document_id", "handover_id"):
                value = _string_value(candidate.get(key))
                if value != "":
                    result[key] = value

        if task.related_type == "document" and task.related_id:
            result.setdefault("document_id", task.related_id)
        if task.related_type == "project" and task.related_id:
            result.setdefault("project_id", task.related_id)
        if task.related_type == "handover" and task.related_id:
            result.setdefault("handover_id", task.related_id)

        return result

    def _validate_scope(self, task: WorkerTask, scope: ObjectDict) -> None:
        if task.related_type == "document" and task.related_id:
            document_id = _string_value(scope.get("document_id"))
            if document_id != "" and document_id != task.related_id:
                raise SkillAdapterError("skill 调用越权：document scope 与任务目标不一致")
        if task.related_type == "project" and task.related_id:
            project_id = _string_value(scope.get("project_id"))
            if project_id != "" and project_id != task.related_id:
                raise SkillAdapterError("skill 调用越权：project scope 与任务目标不一致")

    def _normalize_output(
        self,
        task: WorkerTask,
        output: ObjectDict,
        *,
        skill_name: str,
        skill_version: str,
        source_scope: ObjectDict,
    ) -> ObjectDict:
        normalized = _clone_dict(output)
        normalized["skill_name"] = skill_name
        normalized["skill_version"] = skill_version
        normalized["source_scope"] = _clone_dict(source_scope)

        conversation_id = _string_value(task.payload.get("conversation_id"))
        if conversation_id != "":
            normalized["conversation_id"] = conversation_id

        memory_sources = _normalize_memory_sources(task.payload.get("memory_sources"))
        if len(memory_sources) > 0:
            normalized["memory_sources"] = memory_sources

        raw_suggestions = normalized.get("suggestions")
        if isinstance(raw_suggestions, list):
            normalized["suggestions"] = _normalize_suggestions(raw_suggestions, source_scope)

        return normalized


def _extract_scope_from_payload(payload: ObjectDict) -> ObjectDict:
    scope = _as_object_dict(payload.get("scope"))
    if scope is not None:
        return _clone_dict(scope)

    result: ObjectDict = {}
    for key in ("project_id", "document_id", "handover_id"):
        value = _string_value(payload.get(key))
        if value != "":
            result[key] = value
    return result


def _extract_scope_from_context(context: ObjectDict) -> ObjectDict:
    for key in ("document_context", "project_context", "handover_context"):
        context_item = _as_object_dict(context.get(key))
        if context_item is None:
            continue
        scope = _as_object_dict(context_item.get("scope"))
        if scope is not None:
            return _clone_dict(scope)
    return {}


def _normalize_memory_sources(raw: object) -> list[ObjectDict]:
    if not isinstance(raw, list):
        return []
    items: list[ObjectDict] = []
    for item in cast(list[object], raw):
        item_dict = _as_object_dict(item)
        if item_dict is not None:
            items.append(_clone_dict(item_dict))
    return items


def _normalize_suggestions(raw: list[object], source_scope: ObjectDict) -> list[ObjectDict]:
    items: list[ObjectDict] = []
    scope_text = json.dumps(source_scope, ensure_ascii=False) if len(source_scope) > 0 else ""
    for item in raw:
        item_dict = _as_object_dict(item)
        if item_dict is None:
            continue
        normalized = _clone_dict(item_dict)
        if scope_text != "" and _string_value(normalized.get("source_scope")) == "":
            normalized["source_scope"] = scope_text
        items.append(normalized)
    return items


def _clone_dict(value: ObjectDict) -> ObjectDict:
    cloned: ObjectDict = {}
    for key, item in value.items():
        cloned[key] = item
    return cloned


def _string_value(value: object) -> str:
    return value if isinstance(value, str) else ""


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None
