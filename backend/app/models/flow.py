from datetime import datetime

from sqlalchemy import DateTime, Enum, ForeignKey, Index, String
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, UUIDPrimaryKeyMixin
from app.models.enums import DocumentStatus


class FlowRecord(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "flow_records"
    __table_args__ = (
        Index("idx_flow_records_document_id", "document_id"),
        Index("idx_flow_records_to_user_id", "to_user_id"),
        Index("idx_flow_records_created_at", "created_at"),
    )

    document_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("documents.id"), nullable=False
    )
    version_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("document_versions.id"))
    from_user_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))
    to_user_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))
    from_status: Mapped[DocumentStatus | None] = mapped_column(
        Enum(DocumentStatus, name="document_status", create_type=False)
    )
    to_status: Mapped[DocumentStatus] = mapped_column(
        Enum(DocumentStatus, name="document_status", create_type=False), nullable=False
    )
    action: Mapped[str] = mapped_column(String(32), nullable=False)
    note: Mapped[str | None] = mapped_column(String(500))
    created_by: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
