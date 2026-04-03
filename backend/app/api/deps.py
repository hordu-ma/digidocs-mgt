from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from app.db.session import get_db


DBSession = Annotated[Session, Depends(get_db)]


def get_current_user() -> dict[str, str]:
    return {
        "id": "00000000-0000-0000-0000-000000000001",
        "username": "admin",
        "display_name": "系统管理员",
        "role": "admin",
    }

