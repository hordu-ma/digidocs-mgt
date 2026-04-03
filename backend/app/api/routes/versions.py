from fastapi import APIRouter, File, Form, UploadFile


router = APIRouter()


@router.post("/documents/{document_id}/versions")
def create_version(
    document_id: str,
    commit_message: str | None = Form(None),
    file: UploadFile = File(...),
) -> dict:
    return {
        "data": {
            "id": "00000000-0000-0000-0000-000000000200",
            "document_id": document_id,
            "version_no": 1,
            "commit_message": commit_message,
            "file_name": file.filename,
        }
    }


@router.get("/documents/{document_id}/versions")
def list_versions(document_id: str) -> dict:
    return {"data": [], "meta": {"document_id": document_id}}


@router.get("/{version_id}")
def get_version(version_id: str) -> dict:
    return {"data": {"id": version_id}}


@router.get("/{version_id}/download")
def download_version(version_id: str) -> dict:
    return {"data": {"id": version_id, "download": "not-implemented"}}


@router.get("/{version_id}/preview")
def preview_version(version_id: str) -> dict:
    return {
        "data": {
            "id": version_id,
            "preview_type": "pdf",
            "preview_url": None,
            "watermark_enabled": True,
        }
    }

