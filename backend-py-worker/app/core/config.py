import os
from dataclasses import dataclass


@dataclass(slots=True)
class Settings:
    worker_name: str = "digidocs-py-worker"
    worker_mode: str = "local"
    openclaw_base_url: str = "http://localhost:8001"
    openclaw_api_key: str = "replace-me"
    callback_base_url: str = "http://localhost:8080"
    callback_token: str = "replace-me"


def load_settings() -> Settings:
    return Settings(
        worker_name=os.getenv("WORKER_NAME", "digidocs-py-worker"),
        worker_mode=os.getenv("WORKER_MODE", "local"),
        openclaw_base_url=os.getenv("OPENCLAW_BASE_URL", "http://localhost:8001"),
        openclaw_api_key=os.getenv("OPENCLAW_API_KEY", "replace-me"),
        callback_base_url=os.getenv("CALLBACK_BASE_URL", "http://localhost:8080"),
        callback_token=os.getenv("CALLBACK_TOKEN", "replace-me"),
    )


settings = load_settings()
