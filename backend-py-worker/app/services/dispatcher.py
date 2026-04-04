import time

from app.clients.callback_client import CallbackClient
from app.clients.openclaw_client import OpenClawClient
from app.core.config import settings
from app.tasks.contracts import TaskResult, WorkerTask


class WorkerDispatcher:
    def __init__(self) -> None:
        self.openclaw_client = OpenClawClient()
        self.callback_client = CallbackClient()

    def describe_startup(self) -> None:
        print(
            f"worker={settings.worker_name} mode={settings.worker_mode} "
            f"openclaw={settings.openclaw_base_url}"
        )

    def run_forever(self) -> None:
        self.describe_startup()
        while True:
            time.sleep(60)

    def handle_task(self, task: WorkerTask) -> TaskResult:
        if task.task_type == "assistant.ask":
            answer = self.openclaw_client.ask(
                question=str(task.payload.get("question", "")),
                scope={
                    "project_id": task.payload.get("project_id"),
                    "document_id": task.payload.get("document_id"),
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
            "document.extract_text",
            "assistant.generate_suggestion",
        }:
            return TaskResult(
                request_id=task.request_id,
                status="completed",
                output={
                    "task_type": task.task_type,
                    "queued": True,
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
