import logging
import time
from typing import Any

from app.clients.backend_context_client import BackendContextClient
from app.clients.callback_client import CallbackClient
from app.clients.openclaw_client import OpenClawClient, OpenClawClientError
from app.clients.task_poller import TaskPollerClient
from app.core.config import settings
from app.services.document_text_extractor import (
    DocumentTextExtractionError,
    extract_text,
)
from app.tasks.contracts import TaskResult, WorkerTask

logger = logging.getLogger(__name__)


class WorkerDispatcher:
    def __init__(self) -> None:
        self.openclaw_client = OpenClawClient()
        self.context_client = BackendContextClient()
        self.callback_client = CallbackClient()
        self.poller = TaskPollerClient()

    def describe_startup(self) -> None:
        print(
            f"worker={settings.worker_name} mode={settings.worker_mode} "
            f"openclaw={settings.openclaw_base_url}"
        )

    def run_forever(self) -> None:
        self.describe_startup()
        logger.info("entering poll loop (interval=%ds)", settings.poll_interval)
        while True:
            try:
                tasks = self.poller.poll()
                for task in tasks:
                    logger.info("processing task=%s request_id=%s", task.task_type, task.request_id)
                    self.handle_and_callback(task)
            except Exception:
                logger.exception("poll loop error")
            time.sleep(settings.poll_interval)

    def handle_task(self, task: WorkerTask) -> TaskResult:
        if task.task_type == "assistant.ask":
            scope = task.payload.get("scope", {})
            if not isinstance(scope, dict):
                scope = {}
            normalized_scope = {
                "project_id": scope.get("project_id", task.payload.get("project_id")),
                "document_id": scope.get("document_id", task.payload.get("document_id")),
            }
            context = self._build_context(task, normalized_scope)
            try:
                answer = self.openclaw_client.ask(
                    question=str(task.payload.get("question", "")),
                    scope=normalized_scope,
                    context=context,
                )
            except OpenClawClientError as exc:
                return TaskResult(
                    request_id=task.request_id,
                    status="failed",
                    error_message=str(exc),
                )
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output=answer,
            )

        if task.task_type == "document.summarize":
            context = self._build_context(task)
            extracted_text = ""
            try:
                extracted_text = self._resolve_document_text(task, context)
            except DocumentTextExtractionError as exc:
                context["document_text_warning"] = str(exc)
            if extracted_text:
                context["document_text"] = extracted_text
            try:
                output = self.openclaw_client.summarize_document(task.request_id, task.payload, context)
            except OpenClawClientError as exc:
                return TaskResult(
                    request_id=task.request_id,
                    status="failed",
                    error_message=str(exc),
                )
            if extracted_text:
                output["extracted_text"] = extracted_text
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output=output,
            )

        if task.task_type == "handover.summarize":
            context = self._build_context(task)
            try:
                output = self.openclaw_client.summarize_handover(task.request_id, task.payload, context)
            except OpenClawClientError as exc:
                return TaskResult(
                    request_id=task.request_id,
                    status="failed",
                    error_message=str(exc),
                )
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output=output,
            )

        if task.task_type == "assistant.generate_suggestion":
            context = self._build_context(task)
            try:
                output = self.openclaw_client.generate_suggestion(task.request_id, task.payload, context)
            except OpenClawClientError as exc:
                return TaskResult(
                    request_id=task.request_id,
                    status="failed",
                    error_message=str(exc),
                )
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output=output,
            )

        if task.task_type == "document.extract_text":
            try:
                extracted_text = self._resolve_document_text(task, self._build_context(task))
            except DocumentTextExtractionError as exc:
                return TaskResult(
                    request_id=task.request_id,
                    status="failed",
                    error_message=str(exc),
                )
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output={
                    "task_type": "document.extract_text",
                    "extracted_text": extracted_text,
                },
            )

        return TaskResult(
            request_id=task.request_id,
            status="failed",
            error_message=f"unsupported task type: {task.task_type}",
        )

    def handle_and_callback(self, task: WorkerTask) -> dict:
        result = self.handle_task(task)
        return self.callback_client.submit_result(result)

    def _build_context(
        self,
        task: WorkerTask,
        scope: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        context: dict[str, Any] = {
            "task_type": task.task_type,
            "request_id": task.request_id,
            "related_type": task.related_type,
            "related_id": task.related_id,
            "payload": task.payload,
        }

        if task.related_type == "document" and task.related_id:
            context["document_context"] = self.context_client.get_document_context(task.related_id)
        elif task.related_type == "handover" and task.related_id:
            context["handover_context"] = self.context_client.get_handover_context(task.related_id)

        normalized_scope = scope or {}
        project_id = _string_value(normalized_scope.get("project_id")) or _string_value(
            task.payload.get("project_id")
        )
        document_id = _string_value(normalized_scope.get("document_id")) or _string_value(
            task.payload.get("document_id")
        )

        if document_id and "document_context" not in context:
            context["document_context"] = self.context_client.get_document_context(document_id)
        if project_id:
            context["project_context"] = self.context_client.get_project_context(project_id)

        return context

    def _resolve_document_text(
        self,
        task: WorkerTask,
        context: dict[str, Any],
    ) -> str:
        document_context = context.get("document_context")
        if isinstance(document_context, dict):
            existing_text = _string_value(document_context.get("extracted_text"))
            if existing_text:
                return existing_text

        version_id = _string_value(task.payload.get("version_id"))
        if version_id == "":
            raise DocumentTextExtractionError("缺少 version_id，无法抽取正文")

        headers, content = self.context_client.download_version_file(version_id)
        file_name = _extract_filename(
            headers.get("content_disposition", ""),
            fallback=_string_value(task.payload.get("file_name")) or f"{version_id}.bin",
        )
        extracted_text = extract_text(file_name, content)
        if extracted_text == "":
            raise DocumentTextExtractionError("未提取到有效正文内容")
        return extracted_text


def _string_value(value: Any) -> str:
    return value if isinstance(value, str) else ""


def _extract_filename(content_disposition: str, fallback: str) -> str:
    match = None
    if content_disposition:
        import re

        match = re.search(r'filename="([^"]+)"', content_disposition)
    if match:
        return match.group(1)
    return fallback
