from datetime import datetime

from sqlalchemy import DateTime, Enum, ForeignKey, Index, String, Text
from sqlalchemy.dialects.postgresql import INET, JSONB, UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, UUIDPrimaryKeyMixin
from app.models.enums import AuditActionType


class AuditEvent(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "audit_events"
    __table_args__ = (
        Index("idx_audit_events_document_id", "document_id"),
        Index("idx_audit_events_user_id", "user_id"),
        Index("idx_audit_events_action_type", "action_type"),
        Index("idx_audit_events_created_at", "created_at"),
    )

    document_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("documents.id"))
    version_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("document_versions.id"))
    user_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))
    action_type: Mapped[AuditActionType] = mapped_column(
        Enum(AuditActionType, name="audit_action_type"), nullable=False
    )
    request_id: Mapped[str | None] = mapped_column(String(64))
    ip_address: Mapped[str | None] = mapped_column(INET)
    terminal_info: Mapped[str | None] = mapped_column(String(255))
    extra_data: Mapped[dict | None] = mapped_column(JSONB)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
