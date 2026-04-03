from uuid import UUID

from pydantic import BaseModel


class HandoverCreateRequest(BaseModel):
    target_user_id: UUID
    receiver_user_id: UUID
    project_id: UUID | None = None
    remark: str | None = None


class HandoverActionRequest(BaseModel):
    note: str | None = None

