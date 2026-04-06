from app.clients.openclaw_client import OpenClawClientError
from app.services.dispatcher import WorkerDispatcher
from app.tasks.contracts import WorkerTask


def test_handle_assistant_ask_task(monkeypatch) -> None:
    dispatcher = WorkerDispatcher()
    captured: dict[str, object] = {}

    monkeypatch.setattr(
        dispatcher.context_client,
        "get_project_context",
        lambda project_id: {"available": True, "scope": {"project_id": project_id}},
    )

    def fake_ask(*, question: str, scope: dict, context: dict) -> dict:
        captured["question"] = question
        captured["scope"] = scope
        captured["context"] = context
        return {"answer": "这是回答"}

    monkeypatch.setattr(dispatcher.openclaw_client, "ask", fake_ask)

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-1",
            task_type="assistant.ask",
            payload={
                "question": "请总结当前文档",
                "project_id": "00000000-0000-0000-0000-000000000020",
            },
        )
    )

    assert result.status == "completed"
    assert result.request_id == "req-1"
    assert result.output["answer"] == "这是回答"
    assert captured["question"] == "请总结当前文档"
    assert captured["scope"] == {
        "project_id": "00000000-0000-0000-0000-000000000020",
        "document_id": None,
    }
    assert "project_context" in captured["context"]


def test_handle_unsupported_task_type() -> None:
    dispatcher = WorkerDispatcher()

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-2",
            task_type="unknown.type",  # type: ignore[arg-type]
            payload={},
        )
    )

    assert result.status == "failed"
    assert "unsupported" in (result.error_message or "")


def test_handle_document_summarize_returns_completed(monkeypatch) -> None:
    dispatcher = WorkerDispatcher()

    monkeypatch.setattr(
        dispatcher.context_client,
        "get_document_context",
        lambda document_id: {"available": True, "scope": {"document_id": document_id}},
    )
    monkeypatch.setattr(
        dispatcher.context_client,
        "download_version_file",
        lambda version_id: (
            {
                "content_type": "text/plain; charset=utf-8",
                "content_disposition": 'attachment; filename="notes.txt"',
            },
            "摘要原文".encode(),
        ),
    )
    monkeypatch.setattr(
        dispatcher.openclaw_client,
        "summarize_document",
        lambda request_id, payload, context: {
            "task_type": "document.summarize",
            "summary_text": "这是测试摘要",
            "suggestions": [
                {
                    "title": "AI 摘要",
                    "content": "这是测试摘要",
                    "suggestion_type": "document_summary",
                }
            ],
        },
    )

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-3",
            task_type="document.summarize",
            related_type="document",
            related_id="doc-1",
            payload={"version_id": "ver-1"},
        )
    )

    assert result.status == "completed"
    assert result.output["task_type"] == "document.summarize"
    assert result.output["summary_text"] == "这是测试摘要"


def test_handle_generate_suggestion_openclaw_error(monkeypatch) -> None:
    dispatcher = WorkerDispatcher()

    monkeypatch.setattr(
        dispatcher.openclaw_client,
        "generate_suggestion",
        lambda request_id, payload, context: (_ for _ in ()).throw(OpenClawClientError("boom")),
    )

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-4",
            task_type="assistant.generate_suggestion",
            related_type="document",
            related_id="doc-2",
            payload={},
        )
    )

    assert result.status == "failed"
    assert result.error_message == "boom"


def test_handle_document_extract_text_task(monkeypatch) -> None:
    dispatcher = WorkerDispatcher()

    monkeypatch.setattr(
        dispatcher.context_client,
        "download_version_file",
        lambda version_id: (
            {
                "content_type": "text/plain; charset=utf-8",
                "content_disposition": 'attachment; filename="notes.txt"',
            },
            "第一行\n第二行".encode(),
        ),
    )

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-5",
            task_type="document.extract_text",
            related_type="document",
            related_id="doc-3",
            payload={"version_id": "ver-2"},
        )
    )

    assert result.status == "completed"
    assert result.output["extracted_text"] == "第一行\n第二行"
