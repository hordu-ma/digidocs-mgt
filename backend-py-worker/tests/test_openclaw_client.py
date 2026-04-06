import io
import json
import urllib.error

import pytest

from app.clients.openclaw_client import OpenClawClient, OpenClawClientError


class _FakeResponse:
    def __init__(self, payload: dict):
        self.payload = json.dumps(payload).encode()

    def read(self) -> bytes:
        return self.payload

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None


def test_ask_returns_parsed_answer(monkeypatch) -> None:
    client = OpenClawClient()

    def fake_urlopen(request, timeout):
        assert request.full_url.endswith("/v1/chat/completions")
        assert timeout == client.timeout_seconds
        payload = json.loads(request.data.decode())
        assert payload["model"] == client.model
        return _FakeResponse(
            {
                "id": "resp-1",
                "model": "openclaw/default",
                "choices": [
                    {
                        "message": {
                            "content": "这是 OpenClaw 的回答",
                        }
                    }
                ],
                "usage": {"total_tokens": 12},
            }
        )

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)

    result = client.ask(
        question="当前文档有哪些风险？",
        scope={"document_id": "doc-1"},
        context={"document_context": {"available": True}},
    )

    assert result["answer"] == "这是 OpenClaw 的回答"
    assert result["request_id"] == "resp-1"
    assert result["usage"]["total_tokens"] == 12


def test_summarize_document_parses_json_code_block(monkeypatch) -> None:
    client = OpenClawClient()

    monkeypatch.setattr(
        client,
        "_post",
        lambda path, payload: {
            "id": "resp-2",
            "model": "openclaw/default",
            "choices": [
                {
                    "message": {
                        "content": """```json
{"summary_text":"基于上下文的摘要","suggestions":[{"title":"整理建议","content":"建议补齐正文抽取链路","suggestion_type":"structure_recommendation","confidence":0.72}]}
```""",
                    }
                }
            ],
        },
    )

    result = client.summarize_document(
        request_id="req-1",
        payload={"version_id": "ver-1"},
        context={"document_context": {"available": True}},
    )

    assert result["summary_text"] == "基于上下文的摘要"
    assert result["suggestions"][0]["confidence"] == 0.72


def test_generate_suggestion_raises_on_http_error(monkeypatch) -> None:
    client = OpenClawClient()
    error = urllib.error.HTTPError(
        url="http://localhost:8001/v1/chat/completions",
        code=401,
        msg="Unauthorized",
        hdrs=None,
        fp=io.BytesIO(b'{"error":"unauthorized"}'),
    )

    def fake_urlopen(request, timeout):
        raise error

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)

    with pytest.raises(OpenClawClientError) as exc_info:
        client.generate_suggestion(
            request_id="req-2",
            payload={},
            context={"project_context": {"available": False}},
        )

    assert "HTTP 401" in str(exc_info.value)
