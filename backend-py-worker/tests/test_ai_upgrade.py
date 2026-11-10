"""2026-11-10 AI upgrade: streaming, JSON mode, map-reduce, retrieval, progress."""

import io
import json
import urllib.error
from typing import cast

import pytest

from app.clients.openclaw_client import OpenClawClient, OpenClawClientError, _normalize_suggestions
from app.core.config import settings
from app.services.dispatcher import WorkerDispatcher
from app.tasks.contracts import WorkerTask

type ObjectDict = dict[str, object]


class _Resp:
    def __init__(self, body: bytes) -> None:
        self.body = body

    def read(self) -> bytes:
        return self.body

    def __iter__(self):
        return iter(io.BytesIO(self.body).readlines())

    def __enter__(self):
        return self

    def __exit__(self, *args) -> None:
        return None


def _completion(content: str) -> _Resp:
    return _Resp(json.dumps({"id": "r", "model": "m", "choices": [{"message": {"content": content}}]}).encode())


def test_stream_reports_growing_text(monkeypatch) -> None:
    events = [
        {"id": "s1", "model": "m", "choices": [{"delta": {"content": "第一"}}]},
        {"choices": [{"delta": {"content": "句。"}}]},
    ]
    body = b"".join(f"data: {json.dumps(e, ensure_ascii=False)}\n\n".encode() for e in events) + b"data: [DONE]\n\n"
    seen: list[str] = []

    def fake_urlopen(request, timeout):
        sent = json.loads(request.data.decode())
        assert sent["stream"] is True and sent["stream_options"] == {"include_usage": True}
        return _Resp(body)

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)
    out = OpenClawClient().ask("q", {}, {}, on_delta=seen.append)
    assert out["answer"] == "第一句。" and seen == ["第一", "第一句。"] and out["request_id"] == "s1"


def test_stream_failure_before_first_token_falls_back(monkeypatch) -> None:
    calls: list[bool] = []

    def fake_urlopen(request, timeout):
        streaming = json.loads(request.data.decode()).get("stream", False)
        calls.append(streaming)
        if streaming:
            raise urllib.error.URLError("reset")
        return _completion("非流式回答")

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)
    out = OpenClawClient().ask("q", {}, {}, on_delta=lambda _: None)
    assert out["answer"] == "非流式回答" and calls == [True, False]


def test_json_mode_falls_back_once_on_400(monkeypatch) -> None:
    formats: list[object] = []

    def fake_urlopen(request, timeout):
        payload = json.loads(request.data.decode())
        formats.append(payload.get("response_format"))
        if "response_format" in payload:
            raise urllib.error.HTTPError(request.full_url, 400, "bad", {}, io.BytesIO(b"format unavailable"))
        return _completion('{"summary_text": "ok", "suggestions": []}')

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)
    client = OpenClawClient()
    client.summarize_document("r", {}, {})
    client.summarize_document("r", {}, {})
    assert formats == [{"type": "json_object"}, None, None], "json mode must be disabled after the first 400"


def test_non_json_reply_gets_one_repair_round(monkeypatch) -> None:
    replies = iter(["好的，摘要如下：没有 JSON", '{"summary_text": "修复后", "suggestions": []}'])
    monkeypatch.setattr("urllib.request.urlopen", lambda request, timeout: _completion(next(replies)))
    assert OpenClawClient().summarize_document("r", {}, {})["summary_text"] == "修复后"


def test_json_wrapped_in_prose_is_extracted(monkeypatch) -> None:
    monkeypatch.setattr(
        "urllib.request.urlopen",
        lambda request, timeout: _completion('结果：{"summary_text": "x", "suggestions": []} 以上'),
    )
    assert OpenClawClient().summarize_document("r", {}, {})["summary_text"] == "x"


def test_long_document_is_summarized_map_reduce(monkeypatch) -> None:
    monkeypatch.setattr(settings, "max_context_chars", 100)
    monkeypatch.setattr(settings, "max_summary_sections", 3)
    prompts: list[str] = []

    def fake_urlopen(request, timeout):
        user = json.loads(request.data.decode())["messages"][1]["content"]
        prompts.append(user)
        if "部分要点" in user and "长文档" in user:
            return _completion("本段要点")
        return _completion('{"summary_text": "全文摘要", "suggestions": []}')

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)
    context: ObjectDict = {"document_text": "字" * 1000, "document_context": {"extracted_text": "字" * 1000}}
    out = OpenClawClient().summarize_document("r", {}, context)
    assert out["summary_basis"] == "map_reduce_sampled"
    assert len(prompts) == 4  # 3 sections + final
    assert "字" * 200 not in prompts[-1], "final prompt must carry section notes, not raw text"
    assert "均匀抽样" in prompts[-1]


def test_short_document_basis_and_metadata_only(monkeypatch) -> None:
    monkeypatch.setattr("urllib.request.urlopen", lambda r, timeout: _completion('{"summary_text": "s", "suggestions": []}'))
    client = OpenClawClient()
    assert client.summarize_document("r", {}, {"document_text": "短"})["summary_basis"] == "full_text"
    assert client.summarize_document("r", {}, {})["summary_basis"] == "metadata_only"


def test_suggested_action_whitelist() -> None:
    raw = [
        {"content": "a", "suggested_action": "Finalize"},
        {"content": "b", "suggested_action": "transfer"},
        {"content": "c", "suggested_action": "archive"},
    ]
    out = _normalize_suggestions(raw, "x", allow_action=True)
    assert [s.get("suggested_action") for s in out] == ["finalize", None, "archive"]
    assert all("suggested_action" not in s for s in _normalize_suggestions(raw, "x"))


def test_project_scope_ask_uses_backend_search(monkeypatch) -> None:
    dispatcher = WorkerDispatcher()
    monkeypatch.setattr(dispatcher.context_client, "get_project_context", lambda pid: {"project": {"id": pid}})
    searched: list[tuple[str, str]] = []

    def fake_search(project_id: str, query: str, document_id: str = "", limit: int = 8):
        searched.append((project_id, query))
        return [{"document_id": "d1", "title": "方案", "version_no": 3, "chunk_no": 2, "content": "奖励函数由距离惩罚构成"}]

    monkeypatch.setattr(dispatcher.context_client, "search_project_text", fake_search)
    captured: dict[str, object] = {}

    def fake_run(task, context, scope=None, on_delta=None):
        captured["context"] = context
        return {"answer": "基于 [片段1]"}

    monkeypatch.setattr(dispatcher.skill_adapter, "run", fake_run)
    result = dispatcher.handle_task(WorkerTask(request_id="q1", task_type="assistant.ask", related_type="project", related_id="p1", payload={"question": "奖励函数怎么设计"}))
    assert result.status == "completed" and searched == [("p1", "奖励函数怎么设计")]
    passages = cast(list[ObjectDict], cast(ObjectDict, captured["context"])["passages"])
    assert passages[0]["title"] == "方案"
    citations = cast(list[ObjectDict], result.output["citations"])
    assert citations[0]["document_id"] == "d1" and citations[0]["version_no"] == 3


def test_progress_reporter_is_throttled(monkeypatch) -> None:
    dispatcher = WorkerDispatcher()
    sent: list[str] = []
    monkeypatch.setattr(dispatcher.callback_client, "submit_progress", lambda rid, text: sent.append(text))
    clock = iter([10.0, 10.2, 10.5, 11.3, 11.4])
    monkeypatch.setattr("app.services.dispatcher.time.monotonic", lambda: next(clock))
    monkeypatch.setattr(settings, "progress_interval", 1.0)
    report = dispatcher._progress_reporter(WorkerTask(request_id="r", task_type="assistant.ask"))
    for text in ["a", "ab", "abc", "abcd", "abcde"]:
        report(text)
    assert sent == ["a", "abcd"]


def test_submit_progress_never_raises(monkeypatch) -> None:
    from app.clients.callback_client import CallbackClient

    def boom(*args, **kwargs):
        raise urllib.error.URLError("down")

    monkeypatch.setattr("urllib.request.urlopen", boom)
    CallbackClient().submit_progress("r", "partial")


def test_stream_midway_failure_is_error(monkeypatch) -> None:
    class _Broken(_Resp):
        def __iter__(self):
            yield b'data: {"choices": [{"delta": {"content": "half"}}]}\n'
            raise ConnectionResetError("cut")

    monkeypatch.setattr("urllib.request.urlopen", lambda r, timeout: _Broken(b""))
    with pytest.raises(OpenClawClientError):
        OpenClawClient().ask("q", {}, {}, on_delta=lambda _: None)


def test_check_gateway(monkeypatch) -> None:
    monkeypatch.setattr("urllib.request.urlopen", lambda r, timeout: _Resp(b"{}"))
    assert OpenClawClient().check_gateway()[0] is True

    def boom(r, timeout):
        raise urllib.error.URLError("timed out")

    monkeypatch.setattr("urllib.request.urlopen", boom)
    ok, detail = OpenClawClient().check_gateway()
    assert ok is False and "timed out" in detail


def test_extra_body_is_merged_without_overriding_request_fields(monkeypatch) -> None:
    monkeypatch.setattr(settings, "llm_extra_body", '{"chat_template_kwargs": {"enable_thinking": false}, "model": "ignored"}')
    seen: list[dict] = []

    def fake_urlopen(request, timeout):
        seen.append(json.loads(request.data.decode()))
        return _completion("ok")

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)
    client = OpenClawClient()
    client.ask("q", {}, {})
    assert seen[0]["chat_template_kwargs"] == {"enable_thinking": False}
    assert seen[0]["model"] == client.model, "request fields must win over extra body"


def test_invalid_extra_body_is_ignored(monkeypatch) -> None:
    monkeypatch.setattr(settings, "llm_extra_body", "not json")
    assert OpenClawClient().extra_body == {}
    monkeypatch.setattr(settings, "llm_extra_body", "[1]")
    assert OpenClawClient().extra_body == {}


def test_summaries_report_token_usage_including_map_steps(monkeypatch) -> None:
    monkeypatch.setattr(settings, "max_context_chars", 100)
    monkeypatch.setattr(settings, "max_summary_sections", 2)

    def fake_urlopen(request, timeout):
        user = json.loads(request.data.decode())["messages"][1]["content"]
        content = "要点" if "长文档" in user else '{"summary_text": "s", "suggestions": []}'
        body = {"id": "r", "model": "m", "choices": [{"message": {"content": content}}],
                "usage": {"prompt_tokens": 10, "completion_tokens": 2, "total_tokens": 12}}
        return _Resp(json.dumps(body).encode())

    monkeypatch.setattr("urllib.request.urlopen", fake_urlopen)
    out = OpenClawClient().summarize_document("r", {}, {"document_text": "字" * 500})
    assert out["usage"] == {"prompt_tokens": 30, "completion_tokens": 6, "total_tokens": 36}
    assert "_usage" not in out


def test_long_document_ask_prefers_backend_hybrid_search(monkeypatch) -> None:
    monkeypatch.setattr(settings, "max_context_chars", 200)
    dispatcher = WorkerDispatcher()
    long_text = "无关内容。" * 200
    monkeypatch.setattr(dispatcher.context_client, "get_document_context", lambda doc, version_id="": {
        "scope": {"document_id": doc, "project_id": "p1"}, "extracted_text": long_text,
        "extracted_text_version_id": "v1", "version_id": "v1", "versions": [{"id": "v1", "version_no": 1}],
        "document": {"title": "方案"}})
    monkeypatch.setattr(dispatcher.context_client, "get_project_context", lambda pid: {})
    calls: list[tuple] = []

    def fake_search(project_id, query, document_id="", limit=8):
        calls.append((project_id, document_id))
        return [{"document_id": "d1", "title": "方案", "version_no": 1, "chunk_no": 7, "content": "回报信号由距离惩罚构成"}]

    monkeypatch.setattr(dispatcher.context_client, "search_project_text", fake_search)
    captured: dict[str, object] = {}

    def fake_run(task, context, scope=None, on_delta=None):
        captured["context"] = context
        return {"answer": "ok"}

    monkeypatch.setattr(dispatcher.skill_adapter, "run", fake_run)
    result = dispatcher.handle_task(WorkerTask(request_id="r", task_type="assistant.ask", related_type="document", related_id="d1", payload={"question": "奖励怎么设计"}))
    assert result.status == "completed" and calls == [("p1", "d1")]
    passages = cast(list[ObjectDict], cast(ObjectDict, captured["context"])["passages"])
    assert passages[0]["content"] == "回报信号由距离惩罚构成"

    # Nothing indexed yet: fall back to local ranking of the document text.
    monkeypatch.setattr(dispatcher.context_client, "search_project_text", lambda *a, **k: [])
    dispatcher.handle_task(WorkerTask(request_id="r2", task_type="assistant.ask", related_type="document", related_id="d1", payload={"question": "奖励怎么设计"}))
    passages = cast(list[ObjectDict], cast(ObjectDict, captured["context"])["passages"])
    assert passages and passages[0]["content"].startswith("无关内容")
