from fastapi import APIRouter


router = APIRouter()


@router.get("")
def list_projects(team_space_id: str | None = None) -> dict:
    return {"data": [], "meta": {"team_space_id": team_space_id}}


@router.get("/{project_id}/folders/tree")
def folder_tree(project_id: str) -> dict:
    return {"data": [], "meta": {"project_id": project_id}}

