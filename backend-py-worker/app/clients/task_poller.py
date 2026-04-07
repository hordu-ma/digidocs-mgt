"""HTTP client for polling pending tasks from the Go backend queue."""

from __future__ import annotations

import json
import logging
import urllib.request
from http.client import HTTPResponse
from typing import cast

from ..core.config import settings
from ..tasks.contracts import TaskType, WorkerTask

logger = logging.getLogger(__name__)


class TaskPollerClient:
    """Polls pending tasks from Go backend via GET /api/v1/internal/poll-tasks."""

    def __init__(self) -> None:
        self.base_url: str = settings.callback_base_url
        self.token: str = settings.callback_token

    def poll(self) -> list[WorkerTask]:
        url = f"{self.base_url}/api/v1/internal/poll-tasks"
        req = urllib.request.Request(url, method="GET")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Accept", "application/json")

        try:
            with cast(HTTPResponse, urllib.request.urlopen(req, timeout=10)) as resp:
                parsed = cast(object, json.loads(resp.read()))
        except Exception:
            logger.debug("poll failed (backend may be offline)")
            return []

        body = _as_object_dict(parsed)
        if body is None:
            return []

        data = body.get("data")
        if not isinstance(data, list):
            return []
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
