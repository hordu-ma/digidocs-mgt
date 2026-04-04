from app.core.config import settings
from app.tasks.contracts import TaskResult


class CallbackClient:
    def __init__(self) -> None:
        self.base_url = settings.callback_base_url
        self.token = settings.callback_token

    def submit_result(self, result: TaskResult) -> dict:
        return {
            "callback_base_url": self.base_url,
            "callback_path": "/api/v1/internal/worker-results",
            "request_id": result.request_id,
            "status": result.status,
        }
