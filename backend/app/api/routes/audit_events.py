from fastapi import APIRouter


router = APIRouter()


@router.get("")
def list_audit_events() -> dict:
    return {"data": [], "meta": {"page": 1, "page_size": 20, "total": 0}}


@router.get("/summary")
def audit_summary(project_id: str | None = None) -> dict:
    return {
        "data": {
            "project_id": project_id,
            "download_count": 0,
            "upload_count": 0,
            "transfer_count": 0,
            "archive_count": 0,
            "top_active_users": [],
        }
    }

