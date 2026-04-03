from fastapi import APIRouter

from app.schemas.flow import FlowActionRequest, TransferRequest


router = APIRouter()


@router.post("/{document_id}/flow/mark-in-progress")
def mark_in_progress(document_id: str, payload: FlowActionRequest) -> dict:
    return {"data": {"document_id": document_id, "action": "mark_in_progress", "note": payload.note}}


@router.post("/{document_id}/flow/transfer")
def transfer(document_id: str, payload: TransferRequest) -> dict:
    return {
        "data": {
            "document_id": document_id,
            "action": "transfer",
            "to_user_id": str(payload.to_user_id),
            "note": payload.note,
        }
    }


@router.post("/{document_id}/flow/accept-transfer")
def accept_transfer(document_id: str, payload: FlowActionRequest) -> dict:
    return {"data": {"document_id": document_id, "action": "accept_transfer", "note": payload.note}}


@router.post("/{document_id}/flow/finalize")
def finalize(document_id: str, payload: FlowActionRequest) -> dict:
    return {"data": {"document_id": document_id, "action": "finalize", "note": payload.note}}


@router.post("/{document_id}/flow/archive")
def archive(document_id: str, payload: FlowActionRequest) -> dict:
    return {"data": {"document_id": document_id, "action": "archive", "note": payload.note}}


@router.post("/{document_id}/flow/unarchive")
def unarchive(document_id: str, payload: FlowActionRequest) -> dict:
    return {"data": {"document_id": document_id, "action": "unarchive", "note": payload.note}}


@router.get("/{document_id}/flows")
def list_flows(document_id: str) -> dict:
    return {"data": [], "meta": {"document_id": document_id}}

