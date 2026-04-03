from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, ConfigDict


class ORMModel(BaseModel):
    model_config = ConfigDict(from_attributes=True)


class UserSummary(ORMModel):
    id: UUID
    display_name: str


class TimestampedResponse(ORMModel):
    created_at: datetime
    updated_at: datetime | None = None

