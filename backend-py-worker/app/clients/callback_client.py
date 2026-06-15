"""HTTP client for posting task results back to the Go backend."""

from __future__ import annotations

import json
import logging
import urllib.request
from typing import cast

from ..core.config import settings
from ..tasks.contracts import TaskResult
from .http_util import fetch

type ObjectDict = dict[str, object]

logger = logging.getLogger(__name__)


class CallbackClient:
    base_url: str
    token: str

    def __init__(self) -> None:
        self.base_url = settings.callback_base_url
        self.token = settings.callback_token

    def submit_result(self, result: TaskResult) -> ObjectDict:
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
            _, _, raw = fetch(req, timeout=10, label="worker-results callback")
            parsed = cast(object, json.loads(raw.decode()))
            parsed_dict = _as_object_dict(parsed)
            if parsed_dict is not None:
                return parsed_dict
        except Exception:
            logger.warning(
                "callback failed for request_id=%s after retries", result.request_id, exc_info=True
            )

        return {
            "callback_base_url": self.base_url,
            "request_id": result.request_id,
            "status": "callback_failed",
        }


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None
