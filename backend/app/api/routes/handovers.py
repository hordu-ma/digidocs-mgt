from fastapi import APIRouter

from app.schemas.handover import HandoverActionRequest, HandoverCreateRequest


router = APIRouter()


@router.post("")
def create_handover(payload: HandoverCreateRequest) -> dict:
    return {"data": {"id": "00000000-0000-0000-0000-000000000300", **payload.model_dump(mode="json")}}


@router.get("")
def list_handovers() -> dict:
    return {"data": []}


@router.get("/{handover_id}")
def get_handover(handover_id: str) -> dict:
    return {"data": {"id": handover_id, "items": []}}


@router.patch("/{handover_id}/items")
def update_handover_items(handover_id: str, payload: dict) -> dict:
    return {"data": {"id": handover_id, "items": payload.get("items", [])}}


@router.post("/{handover_id}/confirm")
def confirm_handover(handover_id: str, payload: HandoverActionRequest) -> dict:
    return {"data": {"id": handover_id, "action": "confirm", "note": payload.note}}


@router.post("/{handover_id}/complete")
def complete_handover(handover_id: str, payload: HandoverActionRequest) -> dict:
    return {"data": {"id": handover_id, "action": "complete", "note": payload.note}}


@router.post("/{handover_id}/cancel")
def cancel_handover(handover_id: str, payload: dict) -> dict:
    return {"data": {"id": handover_id, "action": "cancel", "reason": payload.get("reason")}}

