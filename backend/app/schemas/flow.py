from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class FlowActionRequest(BaseModel):
    note: str | None = None


class TransferRequest(FlowActionRequest):
    to_user_id: UUID


class FlowItem(BaseModel):
    id: UUID
    action: str
    from_status: str | None = None
    to_status: str
    created_at: datetime

