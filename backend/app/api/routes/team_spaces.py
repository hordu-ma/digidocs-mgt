from fastapi import APIRouter


router = APIRouter()


@router.get("")
def list_team_spaces() -> dict:
    return {"data": []}

