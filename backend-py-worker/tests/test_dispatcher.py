from app.services.dispatcher import WorkerDispatcher
from app.tasks.contracts import WorkerTask


def test_handle_assistant_ask_task() -> None:
    dispatcher = WorkerDispatcher()

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
    assert "question" in result.output


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


def test_handle_document_summarize_returns_completed() -> None:
    dispatcher = WorkerDispatcher()

    result = dispatcher.handle_task(
        WorkerTask(
            request_id="req-3",
            task_type="document.summarize",
            related_type="document",
            related_id="doc-1",
            payload={"content": "some text"},
        )
    )

    assert result.status == "completed"
    assert result.output["task_type"] == "document.summarize"
