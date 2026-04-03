from app.core.config import settings


class AssistantClient:
    def __init__(self) -> None:
        self.base_url = settings.openclaw_base_url
        self.api_key = settings.openclaw_api_key

    def ask(self, question: str, scope: dict) -> dict:
        return {
            "request_id": "mock-request",
            "question": question,
            "scope": scope,
            "answer": "",
        }

