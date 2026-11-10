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
    openclaw_timeout_seconds: int = 180
    openclaw_skills_assistant_ask: str = "answer_with_context"
    openclaw_skills_document_summarize: str = "document_summary"
    openclaw_skills_handover_summarize: str = "handover_summary"
    openclaw_skills_generate_suggestion: str = "structured_suggestion"
    callback_base_url: str = "http://localhost:8080"
    callback_token: str = "replace-me"
    poll_interval: int = 5
    http_retry_attempts: int = 3
    http_retry_base_delay: float = 0.5
    worker_concurrency: int = 3
    # 2026-11-10: prompt budget and streaming.
    max_context_chars: int = 12000
    max_summary_sections: int = 6
    stream_answers: bool = True
    progress_interval: float = 1.0
    # JSON object merged into every chat completion request, for options a
    # direct OpenAI-compatible model server needs, e.g. Qwen on llama.cpp:
    # {"chat_template_kwargs": {"enable_thinking": false}}
    llm_extra_body: str = ""


def load_settings() -> Settings:
    return Settings(
        worker_name=os.getenv("WORKER_NAME", "digidocs-py-worker"),
        worker_mode=os.getenv("WORKER_MODE", "local"),
        openclaw_base_url=os.getenv("OPENCLAW_BASE_URL", "http://localhost:18789"),
        openclaw_api_key=os.getenv("OPENCLAW_API_KEY", "replace-me"),
        openclaw_model=os.getenv("OPENCLAW_MODEL", "openclaw/default"),
        openclaw_backend_model=os.getenv("OPENCLAW_BACKEND_MODEL", ""),
        openclaw_timeout_seconds=int(os.getenv("OPENCLAW_TIMEOUT_SECONDS", "180")),
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
        http_retry_attempts=int(os.getenv("HTTP_RETRY_ATTEMPTS", "3")),
        http_retry_base_delay=float(os.getenv("HTTP_RETRY_BASE_DELAY", "0.5")),
        worker_concurrency=int(os.getenv("WORKER_CONCURRENCY", "3")),
        max_context_chars=int(os.getenv("MAX_CONTEXT_CHARS", "12000")),
        max_summary_sections=int(os.getenv("MAX_SUMMARY_SECTIONS", "6")),
        stream_answers=os.getenv("STREAM_ANSWERS", "true").lower() == "true",
        progress_interval=float(os.getenv("PROGRESS_INTERVAL", "1.0")),
        llm_extra_body=os.getenv("LLM_EXTRA_BODY", ""),
    )


settings = load_settings()
