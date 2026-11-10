import logging
import os
import re
import signal
import tempfile
import time
from collections.abc import Callable
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from types import FrameType
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
from ..services.context_budget import Passage, passages_from_hits, select_passages
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
        self._stop_requested = False
        self._heartbeat_path = Path(
            os.getenv("WORKER_HEARTBEAT_FILE", str(Path(tempfile.gettempdir()) / "digidocs-worker-heartbeat"))
        )

    def _touch_heartbeat(self) -> None:
        """Record that the poll loop is still turning.

        The worker exposes no port, so a container healthcheck has nothing to
        probe: `restart: unless-stopped` only catches a crashed process, not one
        that is alive but wedged (a hung HTTP call, a deadlock). Stamping a file
        on every loop turn gives the healthcheck something to measure staleness
        against — see the worker healthcheck in docker-compose.yml.
        """
        try:
            self._heartbeat_path.write_text(str(int(time.time())), encoding="utf-8")
        except OSError:
            # A heartbeat failure must never take the worker down; losing the
            # liveness signal is strictly better than losing the worker.
            logger.warning("failed to write heartbeat file %s", self._heartbeat_path, exc_info=True)

    def describe_startup(self) -> None:
        print(
            f"worker={settings.worker_name} mode={settings.worker_mode} openclaw={settings.openclaw_base_url}"
        )

    def _handle_stop_signal(self, signum: int, _frame: FrameType | None) -> None:
        logger.info("received signal %s, will stop after the in-flight batch", signum)
        self._stop_requested = True

    def poll_once(self) -> None:
        """Poll and process exactly one batch, then return. Used by
        WORKER_MODE=once so that mode actually does what its name implies,
        instead of only printing startup info and exiting immediately."""
        try:
            tasks = self.poller.poll()
        except Exception:
            logger.exception("poll failed")
            tasks = []
        with ThreadPoolExecutor(max_workers=max(1, settings.worker_concurrency)) as executor:
            self._process_batch(tasks, executor)

    def run_forever(self) -> None:
        self.describe_startup()
        logger.info(
            "entering poll loop (interval=%ds, concurrency=%d)",
            settings.poll_interval,
            settings.worker_concurrency,
        )
        # Docker's default `docker stop` sends SIGTERM and only escalates to
        # SIGKILL after a grace period; without a handler, a redeploy/restart
        # can kill the process mid-task, leaving it stuck in `running` until
        # the backend's ~10 minute stuck-task recovery kicks in. Checking the
        # flag at the loop boundary (below) lets an in-flight batch finish
        # and skips starting a new one, at the cost of up to poll_interval
        # seconds of shutdown latency.
        signal.signal(signal.SIGTERM, self._handle_stop_signal)
        signal.signal(signal.SIGINT, self._handle_stop_signal)
        self._touch_heartbeat()
        # One pool for the process lifetime rather than one per batch: task
        # handling is I/O-bound (OpenClaw, backend callbacks) or a bounded
        # subprocess (OCR), so there is nothing to gain from tearing threads
        # down between polls, and it avoids paying thread-creation cost every
        # poll_interval.
        with ThreadPoolExecutor(max_workers=max(1, settings.worker_concurrency)) as executor:
            while not self._stop_requested:
                self._touch_heartbeat()
                try:
                    tasks = self.poller.poll()
                except Exception:
                    logger.exception("poll failed")
                    tasks = []
                self._process_batch(tasks, executor)
                if self._stop_requested:
                    break
                time.sleep(settings.poll_interval)
        logger.info("poll loop stopped")

    def _process_batch(self, tasks: list[WorkerTask], executor: ThreadPoolExecutor) -> None:
        """Run a batch through the pool and wait for it to fully drain before
        returning, so the caller's next poll only ever sees tasks that were
        not already claimed by this batch. _process_one already catches and
        reports every exception itself, so .result() here only blocks for
        completion — it is not expected to raise.
        """
        futures = [executor.submit(self._process_one, task) for task in tasks]
        for future in futures:
            future.result()

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
            normalized_scope = _as_object_dict(context.get("scope")) or {}
            # Inline document text extraction when scoped to a document
            if _string_value(normalized_scope.get("document_id")):
                self._ensure_document_text(task, context)
            citations = self._attach_passages(task, context, normalized_scope)
            output, failure = self._run_skill(
                task, context, normalized_scope, on_delta=self._progress_reporter(task)
            )
            if failure:
                return failure
            output["citations"] = citations
            return self._completed(task, output)

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
                output["version_id"] = context.get("resolved_version_id", "")
            return self._completed(task, output)

        if task.task_type in ("handover.summarize", "assistant.generate_suggestion"):
            output, failure = self._run_skill(task, self._build_context(task))
            return failure or self._completed(task, output)

        if task.task_type == "document.extract_text":
            context = self._build_context(task)
            try:
                extracted_text = self._resolve_document_text(task, context)
            except DocumentTextExtractionError as exc:
                return self._failed(task, str(exc))
            return self._completed(
                task,
                {
                    "task_type": "document.extract_text",
                    "extracted_text": extracted_text,
                    "version_id": context.get("resolved_version_id", ""),
                },
            )

        return self._failed(task, f"unsupported task type: {task.task_type}")

    def _attach_passages(
        self, task: WorkerTask, context: ObjectDict, scope: ObjectDict
    ) -> list[ObjectDict]:
        """Replace raw document text in the prompt with budgeted passages.

        Document scope: rank the document's own chunks against the question.
        Project scope: ask the backend to search the project's indexed text.
        Returns citations ([{label, document_id, title, version_no, ...}]) for
        the answer; the model is told to cite passages as [片段N].
        """
        question = _string_value(task.payload.get("question"))
        budget = settings.max_context_chars
        document_id = _string_value(scope.get("document_id"))
        project_id = _string_value(scope.get("project_id"))
        passages: list[Passage] = []
        if document_id:
            text = _string_value(context.pop("document_text", ""))
            doc_ctx = _as_object_dict(context.get("document_context"))
            if doc_ctx is not None:
                text = text or _string_value(doc_ctx.get("extracted_text"))
                doc_ctx.pop("extracted_text", None)
                document = _as_object_dict(doc_ctx.get("document")) or {}
                title = _string_value(document.get("title"))
                version_no = _current_version_no(doc_ctx)
            else:
                title, version_no = "", 0
            passages = select_passages(
                text, question, budget, document_id=document_id, title=title, version_no=version_no
            )
        elif project_id and question:
            hits = self.context_client.search_project_text(project_id, question)
            passages = passages_from_hits(hits, budget)
        context["passages"] = [p.as_context() for p in passages]
        if not passages:
            context["passages_note"] = "未检索到相关正文片段，只能基于元数据回答。"
        return [p.as_citation() for p in passages]

    def _progress_reporter(self, task: WorkerTask) -> Callable[[str], None]:
        """Throttled callback that posts the answer streamed so far."""
        last_sent = 0.0

        def report(text: str) -> None:
            nonlocal last_sent
            now = time.monotonic()
            if now - last_sent < settings.progress_interval:
                return
            last_sent = now
            self.callback_client.submit_progress(task.request_id, text)

        return report

    def _run_skill(
        self,
        task: WorkerTask,
        context: ObjectDict,
        scope: ObjectDict | None = None,
        on_delta: Callable[[str], None] | None = None,
    ) -> tuple[ObjectDict, TaskResult | None]:
        """Run the skill adapter, returning (output, None) on success or
        ({}, failure_result) when the adapter or OpenClaw client errors."""
        try:
            if scope is None:
                output = self.skill_adapter.run(task, context)
            elif on_delta is None:
                output = self.skill_adapter.run(task, context, scope)
            else:
                output = self.skill_adapter.run(task, context, scope, on_delta=on_delta)
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

        # Queued resource identity is authoritative, never arbitrary payload IDs.
        normalized_scope: ObjectDict = {}
        if task.related_type == "document" and task.related_id:
            normalized_scope["document_id"] = task.related_id
        elif task.related_type == "project" and task.related_id:
            normalized_scope["project_id"] = task.related_id
        elif task.related_type == "handover" and task.related_id:
            context["handover_context"] = self.context_client.get_handover_context(task.related_id)
        elif task.task_type == "assistant.ask":
            normalized_scope = scope or {}

        document_id = _string_value(normalized_scope.get("document_id"))
        project_id = _string_value(normalized_scope.get("project_id"))
        if document_id:
            version_id = _string_value(task.payload.get("version_id"))
            doc_context = self.context_client.get_document_context(document_id, version_id)
            if doc_context.get("available") is False:
                raise DocumentTextExtractionError("文档或版本上下文不可用")
            context["document_context"] = doc_context
            document_scope = _as_object_dict(doc_context.get("scope")) or {}
            project_id = _string_value(document_scope.get("project_id"))
            normalized_scope = {"document_id": document_id}
            if project_id:
                normalized_scope["project_id"] = project_id
        if project_id:
            context["project_context"] = self.context_client.get_project_context(project_id)
        context["scope"] = normalized_scope

        return context

    def _ensure_document_text(self, task: WorkerTask, context: ObjectDict) -> None:
        """If document is in scope but extracted_text is empty, try inline extraction."""
        doc_ctx = _as_object_dict(context.get("document_context"))
        if doc_ctx is None:
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
        document_context = _as_object_dict(context.get("document_context")) or {}
        version_id = _string_value(task.payload.get("version_id"))
        if not version_id:
            version_id = _string_value(document_context.get("version_id")) or _latest_version_id(document_context)
        if not version_id:
            raise DocumentTextExtractionError("缺少 version_id，无法抽取正文")
        verified_id = _string_value(document_context.get("version_id"))
        versions = document_context.get("versions")
        known_ids = {
            item.get("id") for item in versions if isinstance(item, dict)
        } if isinstance(versions, list) else set()
        if version_id != verified_id and version_id not in known_ids:
            raise DocumentTextExtractionError("版本不属于当前文档")
        context["resolved_version_id"] = version_id
        if document_context.get("extracted_text_version_id") == version_id:
            existing_text = _string_value(document_context.get("extracted_text"))
            if existing_text:
                return existing_text

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


def _current_version_no(document_context: ObjectDict) -> int:
    version_id = _string_value(document_context.get("version_id")) or _latest_version_id(document_context)
    versions = document_context.get("versions")
    if isinstance(versions, list):
        for item in versions:
            if isinstance(item, dict) and item.get("id") == version_id:
                number = item.get("version_no")
                return number if isinstance(number, int) else 0
    return 0


def _latest_version_id(document_context: ObjectDict | None) -> str:
    """Extract the latest version ID from document_context.versions list."""
    if document_context is None:
        return ""
    versions = document_context.get("versions")
    if not isinstance(versions, list) or not versions:
        return ""
    candidates = [item for item in versions if isinstance(item, dict) and item.get("id")]
    if not candidates:
        return ""
    latest = max(candidates, key=lambda item: int(item.get("version_no") or 0))
    return _string_value(latest.get("id"))


def _extract_filename(content_disposition: str, fallback: str) -> str:
    match = re.search(r'filename="([^"]+)"', content_disposition) if content_disposition else None
    if match:
        return match.group(1)
    return fallback
