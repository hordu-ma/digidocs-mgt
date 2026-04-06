"""HTTP client for posting task results back to the Go backend."""

from __future__ import annotations

import json
import logging
import urllib.request
from typing import Any

from app.core.config import settings
from app.tasks.contracts import TaskResult

logger = logging.getLogger(__name__)


class CallbackClient:
    base_url: str
    token: str

    def __init__(self) -> None:
        self.base_url = settings.callback_base_url
        self.token = settings.callback_token

    def submit_result(self, result: TaskResult) -> dict[str, Any]:
        url = f"{self.base_url}/api/v1/internal/worker-results"
        payload = json.dumps(
            {
                "request_id": result.request_id,
                "status": result.status,
                "output": result.output,
                "error_message": result.error_message,
            }
        ).encode()

        req = urllib.request.Request(url, data=payload, method="POST")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Content-Type", "application/json")

        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                body: str = resp.read().decode()
                return json.loads(body)  # type: ignore[no-any-return]
        except Exception:
            logger.warning("callback failed for request_id=%s", result.request_id)
            return {
                "callback_base_url": self.base_url,
                "request_id": result.request_id,
                "status": "callback_failed",
            }
