"""Shared test configuration."""

from app.core.config import settings

# Keep retry logic exercised but instant in tests (no real backoff sleeps).
settings.http_retry_base_delay = 0.0
