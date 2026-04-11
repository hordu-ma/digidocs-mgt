import os
from dataclasses import dataclass


@dataclass(slots=True)
class Settings:
    worker_name: str = "digidocs-py-worker"
    worker_mode: str = "local"
    openclaw_base_url: str = "http://localhost:18789"
    openclaw_api_key: str = "replace-me"
    openclaw_model: str = "openclaw/default"
    openclaw_backend_model: str = ""
    openclaw_timeout_seconds: int = 60
    openclaw_skills_assistant_ask: str = "answer_with_context"
    openclaw_skills_document_summarize: str = "document_summary"
    openclaw_skills_handover_summarize: str = "handover_summary"
    openclaw_skills_generate_suggestion: str = "structured_suggestion"
    callback_base_url: str = "http://localhost:8080"
    callback_token: str = "replace-me"
    poll_interval: int = 5


def load_settings() -> Settings:
    return Settings(
        worker_name=os.getenv("WORKER_NAME", "digidocs-py-worker"),
        worker_mode=os.getenv("WORKER_MODE", "local"),
        openclaw_base_url=os.getenv("OPENCLAW_BASE_URL", "http://localhost:18789"),
        openclaw_api_key=os.getenv("OPENCLAW_API_KEY", "replace-me"),
        openclaw_model=os.getenv("OPENCLAW_MODEL", "openclaw/default"),
        openclaw_backend_model=os.getenv("OPENCLAW_BACKEND_MODEL", ""),
        openclaw_timeout_seconds=int(os.getenv("OPENCLAW_TIMEOUT_SECONDS", "60")),
        openclaw_skills_assistant_ask=os.getenv(
            "OPENCLAW_SKILLS_ASSISTANT_ASK",
            "answer_with_context",
        ),
        openclaw_skills_document_summarize=os.getenv(
            "OPENCLAW_SKILLS_DOCUMENT_SUMMARIZE",
            "document_summary",
        ),
        openclaw_skills_handover_summarize=os.getenv(
            "OPENCLAW_SKILLS_HANDOVER_SUMMARIZE",
            "handover_summary",
        ),
        openclaw_skills_generate_suggestion=os.getenv(
            "OPENCLAW_SKILLS_GENERATE_SUGGESTION",
            "structured_suggestion",
        ),
        callback_base_url=os.getenv("CALLBACK_BASE_URL", "http://localhost:8080"),
        callback_token=os.getenv("CALLBACK_TOKEN", "replace-me"),
        poll_interval=int(os.getenv("POLL_INTERVAL", "5")),
    )


settings = load_settings()
