from datetime import datetime

from sqlalchemy import Boolean, DateTime, Enum, ForeignKey, Index, String, Text
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin, UUIDPrimaryKeyMixin
from app.models.enums import DocumentStatus


class Document(UUIDPrimaryKeyMixin, TimestampMixin, Base):
    __tablename__ = "documents"
    __table_args__ = (
        Index("idx_documents_project_id", "project_id"),
        Index("idx_documents_folder_id", "folder_id"),
        Index("idx_documents_owner_id", "current_owner_id"),
        Index("idx_documents_status", "current_status"),
        Index("idx_documents_project_status", "project_id", "current_status"),
        Index("idx_documents_updated_at", "updated_at"),
    )

    team_space_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("team_spaces.id"), nullable=False
    )
    project_id: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("projects.id"), nullable=False)
    folder_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("folders.id"))
    title: Mapped[str] = mapped_column(String(255), nullable=False)
    description: Mapped[str | None] = mapped_column(Text)
    file_type: Mapped[str | None] = mapped_column(String(32))
    current_owner_id: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    current_status: Mapped[DocumentStatus] = mapped_column(
        Enum(DocumentStatus, name="document_status"), nullable=False
    )
    current_version_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True))
    is_archived: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    is_deleted: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    created_by: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    deleted_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    deleted_by: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))


class DocumentVersion(UUIDPrimaryKeyMixin, Base):
    __tablename__ = "document_versions"
    __table_args__ = (
        Index("idx_document_versions_document_id", "document_id"),
        Index("idx_document_versions_created_at", "created_at"),
        Index("idx_document_versions_summary_status", "summary_status"),
    )

    document_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("documents.id"), nullable=False
    )
    version_no: Mapped[int] = mapped_column(nullable=False)
    file_name: Mapped[str] = mapped_column(String(255), nullable=False)
    mime_type: Mapped[str | None] = mapped_column(String(128))
    file_size: Mapped[int] = mapped_column(nullable=False)
    storage_provider: Mapped[str] = mapped_column(String(32), nullable=False)
    storage_bucket_or_share: Mapped[str | None] = mapped_column(String(255))
    storage_object_key: Mapped[str] = mapped_column(String(1024), nullable=False)
    external_file_id: Mapped[str | None] = mapped_column(String(255))
    external_path: Mapped[str | None] = mapped_column(String(1024))
    commit_message: Mapped[str | None] = mapped_column(String(500))
    extracted_text_status: Mapped[str] = mapped_column(String(16), nullable=False, default="pending")
    summary_status: Mapped[str] = mapped_column(String(16), nullable=False, default="pending")
    summary_text: Mapped[str | None] = mapped_column(Text)
    created_by: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
