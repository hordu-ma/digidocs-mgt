import logging
import re
import time
from typing import cast

from ..clients.backend_context_client import BackendContextClient
from ..clients.callback_client import CallbackClient
from ..clients.openclaw_client import OpenClawClient, OpenClawClientError
from ..clients.task_poller import TaskPollerClient
from ..core.config import settings
from ..services.document_text_extractor import (
    DocumentTextExtractionError,
    extract_text,
)
from ..services.skill_adapter import SkillAdapterError, WorkerSkillAdapter
from ..tasks.contracts import TaskResult, WorkerTask

type ObjectDict = dict[str, object]

logger = logging.getLogger(__name__)


class WorkerDispatcher:
    def __init__(self) -> None:
        self.openclaw_client: OpenClawClient = OpenClawClient()
        self.context_client: BackendContextClient = BackendContextClient()
        self.callback_client: CallbackClient = CallbackClient()
        self.poller: TaskPollerClient = TaskPollerClient()
        self.skill_adapter: WorkerSkillAdapter = WorkerSkillAdapter(self.openclaw_client)

    def describe_startup(self) -> None:
        print(
            f"worker={settings.worker_name} mode={settings.worker_mode} openclaw={settings.openclaw_base_url}"
        )

    def run_forever(self) -> None:
        self.describe_startup()
        logger.info("entering poll loop (interval=%ds)", settings.poll_interval)
        while True:
            try:
                tasks = self.poller.poll()
            except Exception:
                logger.exception("poll failed")
                tasks = []
            # Isolate each task: one task crashing must not skip its siblings.
            for task in tasks:
                self._process_one(task)
            time.sleep(settings.poll_interval)

    def _process_one(self, task: WorkerTask) -> None:
        started_at = time.perf_counter()
        logger.info("processing task=%s request_id=%s", task.task_type, task.request_id)
        try:
            self.handle_and_callback(task)
        except Exception:
            logger.exception(
                "task crashed task=%s request_id=%s", task.task_type, task.request_id
            )
            # Best-effort: report failure so the request does not hang forever.
            try:
                self.callback_client.submit_result(self._failed(task, "worker internal error"))
            except Exception:
                logger.exception("failed to report crash request_id=%s", task.request_id)
        finally:
            logger.info(
                "task done task=%s request_id=%s duration_ms=%d",
                task.task_type,
                task.request_id,
                int((time.perf_counter() - started_at) * 1000),
            )

    def handle_task(self, task: WorkerTask) -> TaskResult:
        if task.task_type == "assistant.ask":
            scope = _as_object_dict(task.payload.get("scope")) or {}
            normalized_scope: ObjectDict = {
                "project_id": scope.get("project_id") or task.payload.get("project_id"),
                "document_id": scope.get("document_id") or task.payload.get("document_id"),
            }
            context = self._build_context(task, normalized_scope)
            # Inline document text extraction when scoped to a document
            if _string_value(normalized_scope.get("document_id")):
                self._ensure_document_text(task, context)
            output, failure = self._run_skill(task, context, normalized_scope)
            return failure or self._completed(task, output)

        if task.task_type == "document.summarize":
            context = self._build_context(task)
            extracted_text = ""
            try:
                extracted_text = self._resolve_document_text(task, context)
            except DocumentTextExtractionError as exc:
                context["document_text_warning"] = str(exc)
            if extracted_text:
                context["document_text"] = extracted_text
            output, failure = self._run_skill(task, context)
            if failure:
                return failure
            if extracted_text:
                output["extracted_text"] = extracted_text
            return self._completed(task, output)

        if task.task_type in ("handover.summarize", "assistant.generate_suggestion"):
            output, failure = self._run_skill(task, self._build_context(task))
            return failure or self._completed(task, output)

        if task.task_type == "document.extract_text":
            try:
                extracted_text = self._resolve_document_text(task, self._build_context(task))
            except DocumentTextExtractionError as exc:
                return self._failed(task, str(exc))
            return self._completed(
                task,
                {
                    "task_type": "document.extract_text",
                    "extracted_text": extracted_text,
                },
            )

        return self._failed(task, f"unsupported task type: {task.task_type}")

    def _run_skill(
        self,
        task: WorkerTask,
        context: ObjectDict,
        scope: ObjectDict | None = None,
    ) -> tuple[ObjectDict, TaskResult | None]:
        """Run the skill adapter, returning (output, None) on success or
        ({}, failure_result) when the adapter or OpenClaw client errors."""
        try:
            if scope is None:
                output = self.skill_adapter.run(task, context)
            else:
                output = self.skill_adapter.run(task, context, scope)
        except (OpenClawClientError, SkillAdapterError) as exc:
            return {}, self._failed(task, str(exc))
        return output, None

    @staticmethod
    def _completed(task: WorkerTask, output: ObjectDict) -> TaskResult:
        return TaskResult(request_id=task.request_id, status="completed", output=output)

    @staticmethod
    def _failed(task: WorkerTask, message: str) -> TaskResult:
        return TaskResult(request_id=task.request_id, status="failed", error_message=message)

    def handle_and_callback(self, task: WorkerTask) -> ObjectDict:
        result = self.handle_task(task)
        logger.info(
            "task finished request_id=%s status=%s upstream=%s model=%s",
            result.request_id,
            result.status,
            result.output.get("request_id") if isinstance(result.output, dict) else None,
            result.output.get("model") if isinstance(result.output, dict) else None,
        )
        return self.callback_client.submit_result(result)

    def _build_context(
        self,
        task: WorkerTask,
        scope: ObjectDict | None = None,
    ) -> ObjectDict:
        context: ObjectDict = {
            "task_type": task.task_type,
            "request_id": task.request_id,
            "related_type": task.related_type,
            "related_id": task.related_id,
            "payload": cast(object, task.payload),
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

    def _ensure_document_text(self, task: WorkerTask, context: ObjectDict) -> None:
        """If document is in scope but extracted_text is empty, try inline extraction."""
        doc_ctx = _as_object_dict(context.get("document_context"))
        if doc_ctx is None:
            return
        existing = _string_value(doc_ctx.get("extracted_text"))
        if existing:
            return
        try:
            text = self._resolve_document_text(task, context)
            doc_ctx["extracted_text"] = text
            context["document_text"] = text
            logger.info("inline extraction ok request_id=%s chars=%d", task.request_id, len(text))
        except DocumentTextExtractionError as exc:
            logger.warning("inline extraction skipped request_id=%s reason=%s", task.request_id, exc)

    def _resolve_document_text(
        self,
        task: WorkerTask,
        context: ObjectDict,
    ) -> str:
        document_context = _as_object_dict(context.get("document_context"))
        if document_context is not None:
            existing_text = _string_value(document_context.get("extracted_text"))
            if existing_text:
                return existing_text

        version_id = _string_value(task.payload.get("version_id"))
        if version_id == "":
            version_id = _latest_version_id(document_context)
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


def _string_value(value: object) -> str:
    return value if isinstance(value, str) else ""


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None


def _latest_version_id(document_context: ObjectDict | None) -> str:
    """Extract the latest version ID from document_context.versions list."""
    if document_context is None:
        return ""
    versions = document_context.get("versions")
    if not isinstance(versions, list) or not versions:
        return ""
    # versions are ordered by version_no; pick the last (latest)
    for candidate in reversed(cast(list[object], versions)):
        d = _as_object_dict(candidate)
        if d is not None:
            vid = _string_value(d.get("id"))
            if vid:
                return vid
    return ""


def _extract_filename(content_disposition: str, fallback: str) -> str:
    match = re.search(r'filename="([^"]+)"', content_disposition) if content_disposition else None
    if match:
        return match.group(1)
    return fallback
