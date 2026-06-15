"""Shared HTTP helper: urlopen with exponential-backoff retry.

Retries on transient network errors (connection refused, timeouts, DNS) and on
HTTP 5xx responses. Does NOT retry on 4xx (those are deterministic client
errors). On exhaustion the last error is raised so callers keep their existing
failure handling.
"""

from __future__ import annotations

import logging
import time
import urllib.error
import urllib.request
from collections.abc import Callable
from http.client import HTTPResponse
from typing import cast

from ..core.config import settings

logger = logging.getLogger(__name__)


class HttpError(Exception):
    """An HTTP response with a non-2xx status. Carries the body so callers can
    surface upstream error detail (e.g. OpenClaw error messages)."""

    def __init__(self, status: int, body: bytes, reason: str) -> None:
        self.status = status
        self.body = body
        self.reason = reason
        super().__init__(f"HTTP {status}: {reason}")


def fetch(
    req: urllib.request.Request,
    *,
    timeout: float,
    label: str,
    attempts: int | None = None,
    base_delay: float | None = None,
    sleep: Callable[[float], None] = time.sleep,
) -> tuple[int, dict[str, str], bytes]:
    """Perform the request with retry, returning (status, headers, body).

    Raises HttpError for non-retryable / exhausted HTTP errors, or the last
    transport exception when all attempts fail.
    """
    max_attempts = attempts if attempts is not None else settings.http_retry_attempts
    delay = base_delay if base_delay is not None else settings.http_retry_base_delay
    max_attempts = max(1, max_attempts)

    last_exc: Exception | None = None
    for attempt in range(1, max_attempts + 1):
        try:
            with cast(HTTPResponse, urllib.request.urlopen(req, timeout=timeout)) as resp:
                status = int(getattr(resp, "status", 200) or 200)
                headers = dict(getattr(resp, "headers", {}) or {})
                return status, headers, resp.read()
        except urllib.error.HTTPError as exc:
            body = exc.read()
            # 4xx are deterministic; do not retry. 5xx may be transient.
            if exc.code < 500:
                raise HttpError(exc.code, body, str(exc.reason)) from exc
            last_exc = HttpError(exc.code, body, str(exc.reason))
            logger.warning(
                "%s HTTP %s (attempt %d/%d)", label, exc.code, attempt, max_attempts
            )
        except Exception as exc:  # URLError, socket timeout, connection refused…
            last_exc = exc
            logger.warning(
                "%s transient error (attempt %d/%d): %s", label, attempt, max_attempts, exc
            )

        if attempt < max_attempts:
            sleep(delay * (2 ** (attempt - 1)))

    assert last_exc is not None
    raise last_exc
