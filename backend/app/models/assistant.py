from datetime import datetime

from sqlalchemy import DateTime, Enum, ForeignKey, Index, Numeric, String, Text
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, UUIDPrimaryKeyMixin
from app.models.enums import SuggestionStatus, SuggestionType


class AssistantSuggestion(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "assistant_suggestions"
    __table_args__ = (
        Index("idx_assistant_suggestions_related", "related_type", "related_id"),
        Index("idx_assistant_suggestions_status", "status"),
        Index("idx_assistant_suggestions_type", "suggestion_type"),
        Index("idx_assistant_suggestions_generated_at", "generated_at"),
    )

    related_type: Mapped[str] = mapped_column(String(32), nullable=False)
    related_id: Mapped[str] = mapped_column(UUID(as_uuid=True), nullable=False)
    suggestion_type: Mapped[SuggestionType] = mapped_column(
        Enum(SuggestionType, name="suggestion_type"), nullable=False
    )
    status: Mapped[SuggestionStatus] = mapped_column(
        Enum(SuggestionStatus, name="suggestion_status"), nullable=False
    )
    title: Mapped[str | None] = mapped_column(String(255))
    content: Mapped[str] = mapped_column(Text, nullable=False)
    source_scope: Mapped[str | None] = mapped_column(String(255))
    confidence: Mapped[float | None] = mapped_column(Numeric(5, 4))
    request_id: Mapped[str | None] = mapped_column(String(64))
    generated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    expires_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    confirmed_by: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))
    confirmed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    dismissed_by: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))
    dismissed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))


class AssistantRequest(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "assistant_requests"

    request_type: Mapped[str] = mapped_column(String(32), nullable=False)
    related_type: Mapped[str | None] = mapped_column(String(32))
    related_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True))
    payload: Mapped[dict | None] = mapped_column(JSONB)
    status: Mapped[str] = mapped_column(String(16), nullable=False)
    error_message: Mapped[str | None] = mapped_column(Text)
    created_by: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    completed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
