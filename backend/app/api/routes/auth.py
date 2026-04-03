from fastapi import APIRouter

from app.schemas.auth import LoginRequest


router = APIRouter()


@router.post("/login")
def login(payload: LoginRequest) -> dict:
    return {
        "data": {
            "access_token": "dev-token",
            "token_type": "Bearer",
            "expires_in": 7200,
            "user": {
                "id": "00000000-0000-0000-0000-000000000001",
                "username": payload.username,
                "display_name": "开发用户",
                "role": "admin",
            },
        }
    }


@router.get("/me")
def me() -> dict:
    return {
        "data": {
            "id": "00000000-0000-0000-0000-000000000001",
            "username": "admin",
            "display_name": "系统管理员",
            "role": "admin",
            "last_login_at": None,
        }
    }


@router.post("/logout")
def logout() -> dict:
    return {"data": {"success": True}}

