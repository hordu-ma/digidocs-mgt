from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class VersionItem(BaseModel):
    id: UUID
    version_no: int
    file_name: str
    summary_status: str
    created_at: datetime

