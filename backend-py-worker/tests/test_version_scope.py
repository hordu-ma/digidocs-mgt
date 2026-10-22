from unittest.mock import Mock

import pytest

from app.clients.backend_context_client import BackendContextClient
from app.services.dispatcher import WorkerDispatcher, _latest_version_id
from app.services.document_text_extractor import DocumentTextExtractionError
from app.tasks.contracts import WorkerTask


def task(version_id=""):
    return WorkerTask(request_id="r", task_type="document.extract_text", related_type="document",
                      related_id="doc-a", payload={"version_id": version_id})


@pytest.mark.parametrize("versions", [
    [{"id": "v2", "version_no": 2}, {"id": "v1", "version_no": 1}],
    [{"id": "v1", "version_no": 1}, {"id": "v2", "version_no": 2}],
])
def test_latest_version_does_not_depend_on_list_order(versions):
    assert _latest_version_id({"versions": versions}) == "v2"


def test_new_version_never_uses_old_or_unversioned_cache():
    dispatcher = WorkerDispatcher()
    dispatcher.context_client = Mock()
    dispatcher.context_client.download_version_file.return_value = (
        {"content_disposition": 'attachment; filename="v2.txt"'}, b"new body")
    for cached_id in ("v1", ""):
        context = {"document_context": {"version_id": "v2", "extracted_text": "old body",
                                        "extracted_text_version_id": cached_id}}
        assert dispatcher._resolve_document_text(task("v2"), context) == "new body"
        assert context["resolved_version_id"] == "v2"
    assert dispatcher.context_client.download_version_file.call_count == 2


def test_matching_version_cache_avoids_download():
    dispatcher = WorkerDispatcher()
    dispatcher.context_client = Mock()
    context = {"document_context": {"version_id": "v2", "extracted_text": "new body",
                                    "extracted_text_version_id": "v2"}}
    assert dispatcher._resolve_document_text(task("v2"), context) == "new body"
    dispatcher.context_client.download_version_file.assert_not_called()


def test_foreign_version_rejected_before_download():
    dispatcher = WorkerDispatcher()
    dispatcher.context_client = Mock()
    with pytest.raises(DocumentTextExtractionError, match="不属于"):
        dispatcher._resolve_document_text(task("foreign"), {"document_context": {
            "version_id": "v2", "versions": [{"id": "v2", "version_no": 2}]}})
    dispatcher.context_client.download_version_file.assert_not_called()


def test_context_cannot_be_expanded_by_payload_project_id():
    dispatcher = WorkerDispatcher()
    dispatcher.context_client = Mock()
    dispatcher.context_client.get_document_context.return_value = {
        "scope": {"document_id": "doc-a", "project_id": "project-a"}, "version_id": "v2"}
    request = task("v2")
    request.payload["project_id"] = "foreign"
    dispatcher._build_context(request)
    dispatcher.context_client.get_document_context.assert_called_once_with("doc-a", "v2")
    dispatcher.context_client.get_project_context.assert_called_once_with("project-a")


def test_context_client_sends_version_and_encodes_query(monkeypatch):
    client = BackendContextClient()
    fetch = Mock(return_value={})
    monkeypatch.setattr(client, "_fetch", fetch)
    client.get_document_context("doc-a", "v2&project_id=foreign")
    assert fetch.call_args.args[0].endswith("?version_id=v2%26project_id%3Dforeign")


def test_extraction_output_identifies_its_version():
    dispatcher = WorkerDispatcher()
    dispatcher.context_client = Mock()
    dispatcher.context_client.get_document_context.return_value = {
        "version_id": "v2", "extracted_text_version_id": "v2", "extracted_text": "new body"}
    result = dispatcher.handle_task(task("v2"))
    assert result.status == "completed"
    assert result.output["version_id"] == "v2"
    assert result.output["extracted_text"] == "new body"
