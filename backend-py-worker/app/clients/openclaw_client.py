from app.core.config import settings


class OpenClawClient:
    def __init__(self) -> None:
        self.base_url = settings.openclaw_base_url
        self.api_key = settings.openclaw_api_key

    def ask(self, question: str, scope: dict) -> dict:
        return {
            "request_id": "mock-openclaw-request",
            "question": question,
            "scope": scope,
            "answer": "",
        }

