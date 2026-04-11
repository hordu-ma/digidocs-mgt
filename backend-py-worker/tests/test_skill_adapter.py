from app.clients.openclaw_client import OpenClawClientError
from app.services.skill_adapter import SkillAdapterError, WorkerSkillAdapter
from app.tasks.contracts import WorkerTask


def test_skill_adapter_rejects_skill_outside_allowlist() -> None:
    adapter = WorkerSkillAdapter(openclaw_client=None)  # type: ignore[arg-type]

    task = WorkerTask(
        request_id="req-allowlist",
        task_type="assistant.ask",
        payload={
            "question": "测试",
            "project_id": "proj-1",
            "skill_name": "document_summary",
        },
    )

    try:
        adapter.run(task, {"payload": task.payload}, {"project_id": "proj-1"})
    except SkillAdapterError as exc:
        assert "白名单" in str(exc)
    else:
        raise AssertionError("expected SkillAdapterError")


def test_skill_adapter_rejects_cross_scope_document_override() -> None:
    adapter = WorkerSkillAdapter(openclaw_client=None)  # type: ignore[arg-type]

    task = WorkerTask(
        request_id="req-scope",
        task_type="document.summarize",
        related_type="document",
        related_id="doc-1",
        payload={
            "scope": {
                "document_id": "doc-2",
            }
        },
    )

    try:
        adapter.run(task, {"payload": task.payload})
    except SkillAdapterError as exc:
        assert "越权" in str(exc)
    else:
        raise AssertionError("expected SkillAdapterError")


def test_skill_adapter_normalizes_skill_metadata_and_suggestion_scope() -> None:
    class _FakeClient:
        def summarize_document(self, request_id, payload, context):
            return {
                "summary_text": "摘要",
                "suggestions": [
                    {
                        "title": "建议",
                        "content": "补齐目录",
                        "suggestion_type": "structure_recommendation",
                    }
                ],
            }

    adapter = WorkerSkillAdapter(openclaw_client=_FakeClient())  # type: ignore[arg-type]
    task = WorkerTask(
        request_id="req-ok",
        task_type="document.summarize",
        related_type="document",
        related_id="doc-1",
        payload={
            "conversation_id": "conv-1",
            "memory_sources": [{"type": "conversation_messages", "count": 2}],
        },
    )

    output = adapter.run(
        task,
        {
            "payload": task.payload,
            "document_context": {"scope": {"document_id": "doc-1"}},
        },
    )

    assert output["skill_name"] == "document_summary"
    assert output["skill_version"] == "v1"
    assert output["conversation_id"] == "conv-1"
    assert output["source_scope"] == {"document_id": "doc-1"}
    assert output["memory_sources"][0]["type"] == "conversation_messages"
    assert output["suggestions"][0]["source_scope"] == '{"document_id": "doc-1"}'


def test_skill_adapter_bubbles_openclaw_errors() -> None:
    class _FakeClient:
        def ask(self, *, question, scope, context):
            raise OpenClawClientError("boom")

    adapter = WorkerSkillAdapter(openclaw_client=_FakeClient())  # type: ignore[arg-type]
    task = WorkerTask(
        request_id="req-err",
        task_type="assistant.ask",
        payload={"question": "测试", "project_id": "proj-1"},
    )

    try:
        adapter.run(task, {"payload": task.payload}, {"project_id": "proj-1"})
    except OpenClawClientError as exc:
        assert str(exc) == "boom"
    else:
        raise AssertionError("expected OpenClawClientError")
