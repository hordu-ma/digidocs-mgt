from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class DocumentCreateResponse(BaseModel):
    id: UUID
    title: str
    current_status: str
    created_at: datetime | None = None


class DocumentListItem(BaseModel):
    id: UUID
    title: str
    current_status: str
    current_version_no: int | None = None
    updated_at: datetime | None = None


class DocumentDetail(BaseModel):
    id: UUID
    title: str
    description: str | None = None
    current_status: str
    current_owner_id: UUID
    current_version_id: UUID | None = None
    is_archived: bool

