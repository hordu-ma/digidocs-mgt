"""HTTP client for polling pending tasks from the Go backend queue."""

from __future__ import annotations

import json
import logging
import urllib.request
from typing import cast

from ..core.config import settings
from ..tasks.contracts import TaskType, WorkerTask
from .http_util import HttpError, fetch

logger = logging.getLogger(__name__)


class TaskPollerClient:
    """Polls pending tasks from Go backend via GET /api/v1/internal/poll-tasks."""

    def __init__(self) -> None:
        self.base_url: str = settings.callback_base_url
        self.token: str = settings.callback_token
        self.last_poll_ok = False

    def poll(self) -> list[WorkerTask]:
        self.last_poll_ok = False
        limit = min(10, max(1, settings.worker_concurrency))
        url = f"{self.base_url}/api/v1/internal/poll-tasks?limit={limit}"
        req = urllib.request.Request(url, method="GET")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Accept", "application/json")

        try:
            _, _, raw = fetch(req, timeout=10, label="poll-tasks")
            parsed = cast(object, json.loads(raw))
        except HttpError as exc:
            if exc.status in (401, 403):
                # Not a transient outage: CALLBACK_TOKEN is wrong/expired, and
                # the worker will silently process zero tasks forever until an
                # operator notices. Must be loud even at default INFO level.
                logger.error(
                    "poll-tasks rejected with HTTP %d: CALLBACK_TOKEN is likely "
                    "invalid or expired; worker will not receive any tasks "
                    "until this is fixed",
                    exc.status,
                )
            else:
                logger.warning("poll-tasks failed with HTTP %d", exc.status)
            return []
        except Exception as exc:
            logger.warning("poll failed (backend may be offline): %s", exc)
            return []

        body = _as_object_dict(parsed)
        if body is None:
            return []

        data = body.get("data")
        if not isinstance(data, list):
            return []
        self.last_poll_ok = True
        items = cast(list[object], data)

        tasks: list[WorkerTask] = []
        for item in items:
            item_dict = _as_object_dict(item)
            if item_dict is None:
                continue

            request_id = item_dict.get("request_id")
            task_type = _parse_task_type(item_dict.get("task_type"))
            if not isinstance(request_id, str) or task_type is None:
                continue

            attempt = item_dict.get("attempt")
            payload = _as_object_dict(item_dict.get("payload")) or {}
            related_type = item_dict.get("related_type")
            related_id = item_dict.get("related_id")

            tasks.append(
                WorkerTask(
                    request_id=request_id,
                    task_type=task_type,
                    related_type=related_type if isinstance(related_type, str) else None,
                    related_id=related_id if isinstance(related_id, str) else None,
                    payload=payload,
                    attempt=attempt if isinstance(attempt, int) else 0,
                )
            )
        return tasks


def _as_object_dict(value: object) -> dict[str, object] | None:
    return cast(dict[str, object], value) if isinstance(value, dict) else None


def _parse_task_type(value: object) -> TaskType | None:
    if value in {
        "assistant.ask",
        "document.summarize",
        "handover.summarize",
        "document.extract_text",
        "assistant.generate_suggestion",
    }:
        return cast(TaskType, value)
    return None
