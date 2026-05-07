import json

from app.clients.backend_context_client import BackendContextClient
from app.clients.callback_client import CallbackClient
from app.clients.task_poller import TaskPollerClient
from app.tasks.contracts import TaskResult


class _FakeResponse:
    def __init__(self, payload: object, headers: dict[str, str] | None = None) -> None:
        self.payload = payload
        self.headers = headers or {}

    def read(self) -> bytes:
        if isinstance(self.payload, bytes):
            return self.payload
        return json.dumps(self.payload).encode()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None


def test_backend_context_fetch_success_and_invalid_response(monkeypatch) -> None:
    client = BackendContextClient()
    calls: list[str] = []

    def fake_urlopen(request, timeout):
        calls.append(request.full_url)
        return _FakeResponse({"data": {"id": "doc-1", "scope": {"document_id": "doc-1"}}})

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)

    context = client.get_document_context("doc-1")

    assert calls[0].endswith("/api/v1/internal/assistant-context/documents/doc-1")
    assert context["available"] is True
    assert context["id"] == "doc-1"

    monkeypatch.setattr("urllib.request.urlopen", lambda request, timeout: _FakeResponse([]))

    context = client.get_project_context("project-1")

    assert context == {
        "available": False,
        "error": "invalid context response",
        "path": "/api/v1/internal/assistant-context/projects/project-1",
    }


def test_backend_context_download_version_file(monkeypatch) -> None:
    client = BackendContextClient()

    monkeypatch.setattr(
        "urllib.request.urlopen",
        lambda request, timeout: _FakeResponse(
            b"file-content",
            headers={
                "Content-Type": "text/plain",
                "Content-Disposition": 'attachment; filename="notes.txt"',
            },
        ),
    )

    headers, content = client.download_version_file("ver-1")

    assert headers == {
        "content_type": "text/plain",
        "content_disposition": 'attachment; filename="notes.txt"',
    }
    assert content == b"file-content"


def test_task_poller_parses_valid_tasks_and_skips_invalid_items(monkeypatch) -> None:
    poller = TaskPollerClient()

    monkeypatch.setattr(
        "urllib.request.urlopen",
        lambda request, timeout: _FakeResponse(
            {
                "data": [
                    {
                        "request_id": "req-1",
                        "task_type": "assistant.ask",
                        "payload": {"question": "测试"},
                        "related_type": "document",
                        "related_id": "doc-1",
                    },
                    {"request_id": "req-unknown", "task_type": "unknown"},
                    {"task_type": "document.summarize"},
                    "invalid",
                ]
            }
        ),
    )

    tasks = poller.poll()

    assert len(tasks) == 1
    assert tasks[0].request_id == "req-1"
    assert tasks[0].task_type == "assistant.ask"
    assert tasks[0].related_id == "doc-1"
    assert tasks[0].payload == {"question": "测试"}


def test_task_poller_returns_empty_list_on_backend_failure(monkeypatch) -> None:
    poller = TaskPollerClient()

    def fake_urlopen(request, timeout):
        raise OSError("offline")

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)

    assert poller.poll() == []


def test_callback_client_returns_backend_response_or_failure(monkeypatch) -> None:
    client = CallbackClient()

    monkeypatch.setattr(
        "urllib.request.urlopen",
        lambda request, timeout: _FakeResponse({"data": {"ok": True}}),
    )

    response = client.submit_result(
        TaskResult(request_id="req-1", status="completed", output={"answer": "ok"})
    )

    assert response == {"data": {"ok": True}}

    def fake_failure(request, timeout):
        raise OSError("backend down")

    monkeypatch.setattr("urllib.request.urlopen", fake_failure)

    response = client.submit_result(TaskResult(request_id="req-2", status="failed"))

    assert response["status"] == "callback_failed"
    assert response["request_id"] == "req-2"
