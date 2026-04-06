"""Internal client for fetching worker-only assistant context from Go backend."""

from __future__ import annotations

import json
import logging
import urllib.request
from typing import Any

from app.core.config import settings

logger = logging.getLogger(__name__)


class BackendContextClient:
    def __init__(self) -> None:
        self.base_url = settings.callback_base_url.rstrip("/")
        self.token = settings.callback_token

    def get_document_context(self, document_id: str) -> dict[str, Any]:
        return self._fetch(f"/api/v1/internal/assistant-context/documents/{document_id}")

    def get_project_context(self, project_id: str) -> dict[str, Any]:
        return self._fetch(f"/api/v1/internal/assistant-context/projects/{project_id}")

    def get_handover_context(self, handover_id: str) -> dict[str, Any]:
        return self._fetch(f"/api/v1/internal/assistant-context/handovers/{handover_id}")

    def _fetch(self, path: str) -> dict[str, Any]:
        url = f"{self.base_url}{path}"
        req = urllib.request.Request(url, method="GET")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Accept", "application/json")

        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                body = json.loads(resp.read())
        except Exception as exc:
            logger.warning("assistant context fetch failed path=%s", path)
            return {
                "available": False,
                "error": str(exc),
                "path": path,
            }

        data = body.get("data")
        if isinstance(data, dict):
            data["available"] = True
            return data

        return {
            "available": False,
            "error": "invalid context response",
            "path": path,
        }
