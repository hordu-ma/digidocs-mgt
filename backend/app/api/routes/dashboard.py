from fastapi import APIRouter


router = APIRouter()


@router.get("/overview")
def overview(project_id: str | None = None) -> dict:
    return {
        "data": {
            "project_id": project_id,
            "document_total": 0,
            "status_counts": {},
            "handover_pending_count": 0,
            "risk_document_count": 0,
        }
    }


@router.get("/recent-flows")
def recent_flows(project_id: str | None = None) -> dict:
    return {"data": [], "meta": {"project_id": project_id}}


@router.get("/risk-documents")
def risk_documents(project_id: str | None = None) -> dict:
    return {"data": [], "meta": {"project_id": project_id}}

