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
        """Post the task result back to the Go backend.

        Raises on failure after retries are exhausted (via fetch()) instead of
        swallowing the error and returning a synthetic "callback_failed" dict:
        the request is already stuck in Go's `running` state and will only be
        recovered ~10 minutes later and reprocessed from scratch (duplicate
        OpenClaw call), so a swallowed failure here previously vanished with
        no trace. Raising lets the caller's existing crash-handling path
        (WorkerDispatcher._process_one) log it loudly via logger.exception.
        """
        url = f"{self.base_url}/api/v1/internal/worker-results"
        payload = json.dumps(
            {
                "request_id": result.request_id,
                "attempt": result.attempt,
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
        except Exception:
            logger.warning(
                "callback failed for request_id=%s after retries", result.request_id, exc_info=True
            )
            raise

        parsed_dict = _as_object_dict(parsed)
        if parsed_dict is not None:
            return parsed_dict
        return {"request_id": result.request_id, "status": result.status}


    def submit_progress(self, request_id: str, partial_answer: str, attempt: int = 0) -> None:
        """Best effort: store the answer streamed so far. One attempt with a
        short timeout and no exception, because a lost progress update only
        delays what the user sees; the final result is posted separately."""
        url = f"{self.base_url}/api/v1/internal/worker-progress"
        payload = json.dumps({"request_id": request_id, "partial_answer": partial_answer, "attempt": attempt}).encode()
        req = urllib.request.Request(url, data=payload, method="POST")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Content-Type", "application/json")
        try:
            fetch(req, timeout=3, label="worker-progress", attempts=1)
        except Exception as exc:
            logger.debug("progress update dropped request_id=%s: %s", request_id, exc)


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None
