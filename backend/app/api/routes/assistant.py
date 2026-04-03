from fastapi import APIRouter

from app.schemas.assistant import AskRequest


router = APIRouter()


@router.post("/ask")
def ask(payload: AskRequest) -> dict:
    return {
        "data": {
            "request_id": "00000000-0000-0000-0000-000000000400",
            "question": payload.question,
            "answer": "",
            "source_scope": {
                "project_id": str(payload.project_id) if payload.project_id else None,
                "document_id": str(payload.document_id) if payload.document_id else None,
            },
        }
    }


@router.post("/documents/{document_id}/summarize")
def summarize_document(document_id: str, payload: dict) -> dict:
    return {"data": {"document_id": document_id, "status": "queued", "payload": payload}}


@router.post("/handovers/{handover_id}/summarize")
def summarize_handover(handover_id: str) -> dict:
    return {"data": {"handover_id": handover_id, "status": "queued"}}


@router.get("/suggestions")
def list_suggestions() -> dict:
    return {"data": []}


@router.post("/suggestions/{suggestion_id}/confirm")
def confirm_suggestion(suggestion_id: str, payload: dict) -> dict:
    return {"data": {"id": suggestion_id, "action": "confirm", "payload": payload}}


@router.post("/suggestions/{suggestion_id}/dismiss")
def dismiss_suggestion(suggestion_id: str, payload: dict) -> dict:
    return {"data": {"id": suggestion_id, "action": "dismiss", "payload": payload}}

