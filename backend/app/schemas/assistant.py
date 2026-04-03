from uuid import UUID

from pydantic import BaseModel


class AskRequest(BaseModel):
    project_id: UUID | None = None
    document_id: UUID | None = None
    question: str

