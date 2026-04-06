import logging
import time

from app.clients.callback_client import CallbackClient
from app.clients.openclaw_client import OpenClawClient
from app.clients.task_poller import TaskPollerClient
from app.core.config import settings
from app.tasks.contracts import TaskResult, WorkerTask

logger = logging.getLogger(__name__)


class WorkerDispatcher:
    def __init__(self) -> None:
        self.openclaw_client = OpenClawClient()
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
            answer = self.openclaw_client.ask(
                question=str(task.payload.get("question", "")),
                scope={
                    "project_id": scope.get("project_id", task.payload.get("project_id")),
                    "document_id": scope.get("document_id", task.payload.get("document_id")),
                },
            )
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output=answer,
            )

        if task.task_type in {
            "document.summarize",
            "handover.summarize",
        }:
            summary_text = (
                "文档摘要任务已完成，当前为占位结果，待真实 OpenClaw 摘要接入。"
                if task.task_type == "document.summarize"
                else "交接摘要任务已完成，当前为占位结果，待真实 OpenClaw 摘要接入。"
            )
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output={
                    "task_type": task.task_type,
                    "summary_text": summary_text,
                    "suggestions": [
                        {
                            "title": "AI 摘要",
                            "content": summary_text,
                            "suggestion_type": (
                                "document_summary"
                                if task.task_type == "document.summarize"
                                else "handover_summary"
                            ),
                        }
                    ],
                },
            )

        if task.task_type in {
            "document.extract_text",
            "assistant.generate_suggestion",
        }:
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output={
                    "task_type": task.task_type,
                    "suggestions": [
                        {
                            "title": "AI 建议",
                            "content": "建议任务已完成，当前为占位结果，待真实 OpenClaw 建议接入。",
                            "suggestion_type": "structure_recommendation",
                        }
                    ],
                    "payload": task.payload,
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
