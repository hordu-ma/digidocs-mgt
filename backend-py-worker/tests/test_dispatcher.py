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

