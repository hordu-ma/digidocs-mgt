"""HTTP client for polling pending tasks from the Go backend queue."""

import json
import logging
import urllib.request

from app.core.config import settings
from app.tasks.contracts import WorkerTask

logger = logging.getLogger(__name__)


class TaskPollerClient:
    """Polls pending tasks from Go backend via GET /api/v1/internal/poll-tasks."""

    def __init__(self) -> None:
        self.base_url = settings.callback_base_url
        self.token = settings.callback_token

    def poll(self) -> list[WorkerTask]:
        url = f"{self.base_url}/api/v1/internal/poll-tasks"
        req = urllib.request.Request(url, method="GET")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Accept", "application/json")

        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                body = json.loads(resp.read())
        except Exception:
            logger.debug("poll failed (backend may be offline)")
            return []

        items = body.get("data", [])
        tasks: list[WorkerTask] = []
        for item in items:
            tasks.append(
                WorkerTask(
                    request_id=item["request_id"],
                    task_type=item["task_type"],
                    related_type=item.get("related_type"),
                    related_id=item.get("related_id"),
                    payload=item.get("payload", {}),
                )
            )
        return tasks
