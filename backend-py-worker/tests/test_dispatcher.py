from typing import cast

from _pytest.monkeypatch import MonkeyPatch

from app.clients.openclaw_client import OpenClawClientError
from app.services.dispatcher import WorkerDispatcher
from app.tasks.contracts import TaskType, WorkerTask


type ObjectDict = dict[str, object]


def test_handle_assistant_ask_task(monkeypatch: MonkeyPatch) -> None:
    dispatcher = WorkerDispatcher()
    captured: dict[str, object] = {}

    monkeypatch.setattr(
        dispatcher.context_client,
        "get_project_context",
        lambda project_id: {"available": True, "scope": {"project_id": project_id}},
    )

    def fake_run(task: WorkerTask, context: ObjectDict, scope: ObjectDict | None = None) -> ObjectDict:
        captured["task"] = task
        captured["scope"] = scope or {}
        captured["context"] = context
        return {
            "answer": "这是回答",
            "skill_name": "answer_with_context",
            "skill_version": "v1",
            "source_scope": scope or {},
        }

    monkeypatch.setattr(dispatcher.skill_adapter, "run", fake_run)

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
    assert captured["scope"] == {
        "project_id": "00000000-0000-0000-0000-000000000020",
        "document_id": None,
    }
    assert "project_context" in cast(ObjectDict, captured["context"])


def test_handle_unsupported_task_type() -> None:
    dispatcher = WorkerDispatcher()

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-2",
            task_type=cast(TaskType, "unknown.type"),
            payload={},
        )
    )

    assert result.status == "failed"
    assert "unsupported" in (result.error_message or "")


def test_handle_document_summarize_returns_completed(monkeypatch: MonkeyPatch) -> None:
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
        dispatcher.skill_adapter,
        "run",
        lambda task, context, scope=None: {
            "task_type": "document.summarize",
            "summary_text": "这是测试摘要",
            "skill_name": "document_summary",
            "skill_version": "v1",
            "source_scope": {"document_id": "doc-1"},
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
    assert result.output["skill_name"] == "document_summary"

def test_handle_generate_suggestion_openclaw_error(monkeypatch: MonkeyPatch) -> None:
    dispatcher = WorkerDispatcher()

    monkeypatch.setattr(
        dispatcher.skill_adapter,
        "run",
        lambda task, context, scope=None: (_ for _ in ()).throw(OpenClawClientError("boom")),
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


def test_handle_document_extract_text_task(monkeypatch: MonkeyPatch) -> None:
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


def test_resolve_document_text_falls_back_to_latest_context_version(monkeypatch: MonkeyPatch) -> None:
    dispatcher = WorkerDispatcher()
    downloaded: dict[str, str] = {}

    def fake_download(version_id: str):
        downloaded["version_id"] = version_id
        return (
            {"content_disposition": 'attachment; filename="latest.txt"'},
            "最新版本正文".encode(),
        )

    monkeypatch.setattr(dispatcher.context_client, "download_version_file", fake_download)

    text = dispatcher._resolve_document_text(
        WorkerTask(
            request_id="req-latest",
            task_type="document.extract_text",
            related_type="document",
            related_id="doc-1",
            payload={},
        ),
        {
            "document_context": {
                "versions": [
                    {"id": "ver-old", "version_no": 1},
                    {"id": "ver-latest", "version_no": 2},
                ]
            }
        },
    )

    assert downloaded["version_id"] == "ver-latest"
    assert text == "最新版本正文"


def test_document_extract_text_reports_empty_extraction(monkeypatch: MonkeyPatch) -> None:
    dispatcher = WorkerDispatcher()
    monkeypatch.setattr(
        dispatcher.context_client,
        "download_version_file",
        lambda version_id: ({"content_disposition": 'attachment; filename="empty.txt"'}, b""),
    )

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-empty",
            task_type="document.extract_text",
            related_type="document",
            related_id="doc-1",
            payload={"version_id": "ver-empty"},
        )
    )

    assert result.status == "failed"
    assert "未提取到有效正文内容" in (result.error_message or "")


def test_assistant_ask_inline_extraction_updates_context(monkeypatch: MonkeyPatch) -> None:
    dispatcher = WorkerDispatcher()
    captured: dict[str, object] = {}

    monkeypatch.setattr(
        dispatcher.context_client,
        "get_document_context",
        lambda document_id: {
            "scope": {"document_id": document_id},
            "extracted_text": "",
            "versions": [{"id": "ver-1", "version_no": 1}],
        },
    )
    monkeypatch.setattr(
        dispatcher.context_client,
        "download_version_file",
        lambda version_id: (
            {"content_disposition": 'attachment; filename="doc.txt"'},
            "内联正文".encode(),
        ),
    )

    def fake_run(task: WorkerTask, context: ObjectDict, scope: ObjectDict | None = None) -> ObjectDict:
        captured["context"] = context
        return {"answer": "ok"}

    monkeypatch.setattr(dispatcher.skill_adapter, "run", fake_run)

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-inline",
            task_type="assistant.ask",
            payload={"question": "总结", "scope": {"document_id": "doc-1"}},
        )
    )

    context = cast(ObjectDict, captured["context"])
    doc_context = cast(ObjectDict, context["document_context"])
    assert result.status == "completed"
    assert context["document_text"] == "内联正文"
    assert doc_context["extracted_text"] == "内联正文"
