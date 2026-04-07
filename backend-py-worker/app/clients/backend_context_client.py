"""Internal client for fetching worker-only assistant context from Go backend."""

from __future__ import annotations

import json
import logging
import urllib.request
from http.client import HTTPResponse
from typing import cast

from ..core.config import settings

type ObjectDict = dict[str, object]

logger = logging.getLogger(__name__)


class BackendContextClient:
    def __init__(self) -> None:
        self.base_url: str = settings.callback_base_url.rstrip("/")
        self.token: str = settings.callback_token

    def get_document_context(self, document_id: str) -> ObjectDict:
        return self._fetch(f"/api/v1/internal/assistant-context/documents/{document_id}")

    def get_project_context(self, project_id: str) -> ObjectDict:
        return self._fetch(f"/api/v1/internal/assistant-context/projects/{project_id}")

    def get_handover_context(self, handover_id: str) -> ObjectDict:
        return self._fetch(f"/api/v1/internal/assistant-context/handovers/{handover_id}")

    def download_version_file(self, version_id: str) -> tuple[dict[str, str], bytes]:
        url = f"{self.base_url}/api/v1/internal/assistant-assets/versions/{version_id}/download"
        req = urllib.request.Request(url, method="GET")
        req.add_header("Authorization", f"Bearer {self.token}")

        with cast(HTTPResponse, urllib.request.urlopen(req, timeout=20)) as resp:
            headers = {
                "content_type": resp.headers.get("Content-Type", ""),
                "content_disposition": resp.headers.get("Content-Disposition", ""),
            }
            return headers, resp.read()

    def _fetch(self, path: str) -> ObjectDict:
        url = f"{self.base_url}{path}"
        req = urllib.request.Request(url, method="GET")
        req.add_header("Authorization", f"Bearer {self.token}")
        req.add_header("Accept", "application/json")

        try:
            with cast(HTTPResponse, urllib.request.urlopen(req, timeout=10)) as resp:
                parsed = cast(object, json.loads(resp.read()))
        except Exception as exc:
            logger.warning("assistant context fetch failed path=%s", path)
            return {
                "available": False,
                "error": str(exc),
                "path": path,
            }

        body = _as_object_dict(parsed)
        if body is None:
            return {
                "available": False,
                "error": "invalid context response",
                "path": path,
            }

        data = body.get("data")
        data_dict = _as_object_dict(data)
        if data_dict is not None:
            data_dict["available"] = True
            return data_dict

        return {
            "available": False,
            "error": "invalid context response",
            "path": path,
        }


def _as_object_dict(value: object) -> ObjectDict | None:
    return cast(ObjectDict, value) if isinstance(value, dict) else None
