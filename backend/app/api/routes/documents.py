from fastapi import APIRouter, File, Form, UploadFile


router = APIRouter()


@router.post("")
def create_document(
    team_space_id: str = Form(...),
    project_id: str = Form(...),
    folder_id: str | None = Form(None),
    title: str = Form(...),
    description: str | None = Form(None),
    current_owner_id: str = Form(...),
    commit_message: str | None = Form(None),
    file: UploadFile = File(...),
) -> dict:
    return {
        "data": {
            "id": "00000000-0000-0000-0000-000000000100",
            "team_space_id": team_space_id,
            "project_id": project_id,
            "folder_id": folder_id,
            "title": title,
            "description": description,
            "current_owner_id": current_owner_id,
            "commit_message": commit_message,
            "file_name": file.filename,
            "current_status": "draft",
        }
    }


@router.get("")
def list_documents() -> dict:
    return {"data": [], "meta": {"page": 1, "page_size": 20, "total": 0}}


@router.get("/{document_id}")
def get_document(document_id: str) -> dict:
    return {"data": {"id": document_id}}


@router.patch("/{document_id}")
def update_document(document_id: str, payload: dict) -> dict:
    return {"data": {"id": document_id, "updated": payload}}


@router.post("/{document_id}/delete")
def delete_document(document_id: str, payload: dict) -> dict:
    return {"data": {"id": document_id, "is_deleted": True, "reason": payload.get("reason")}}


@router.post("/{document_id}/restore")
def restore_document(document_id: str) -> dict:
    return {"data": {"id": document_id, "is_deleted": False}}

