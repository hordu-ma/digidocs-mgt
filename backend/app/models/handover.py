from datetime import datetime

from sqlalchemy import DateTime, Enum, ForeignKey, Index, String, Text, UniqueConstraint
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, UUIDPrimaryKeyMixin
from app.models.enums import HandoverStatus


class GraduationHandover(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "graduation_handovers"
    __table_args__ = (
        Index("idx_graduation_handovers_target_user_id", "target_user_id"),
        Index("idx_graduation_handovers_receiver_user_id", "receiver_user_id"),
        Index("idx_graduation_handovers_status", "status"),
    )

    target_user_id: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    receiver_user_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("users.id"), nullable=False
    )
    project_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("projects.id"))
    status: Mapped[HandoverStatus] = mapped_column(
        Enum(HandoverStatus, name="handover_status"), nullable=False
    )
    remark: Mapped[str | None] = mapped_column(String(500))
    ai_summary: Mapped[str | None] = mapped_column(Text)
    generated_by: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    generated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    confirmed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    completed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    cancelled_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))


class GraduationHandoverItem(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "graduation_handover_items"
    __table_args__ = (
        UniqueConstraint("handover_id", "document_id", name="uq_handover_items_handover_document"),
    )

    handover_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("graduation_handovers.id"), nullable=False
    )
    document_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("documents.id"), nullable=False
    )
    selected: Mapped[bool] = mapped_column(nullable=False, default=True)
    note: Mapped[str | None] = mapped_column(String(500))
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
