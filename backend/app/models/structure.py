from sqlalchemy import ForeignKey, Index, String, Text, UniqueConstraint
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin, UUIDPrimaryKeyMixin


class TeamSpace(UUIDPrimaryKeyMixin, TimestampMixin, Base):
    __tablename__ = "team_spaces"

    name: Mapped[str] = mapped_column(String(128), unique=True, nullable=False)
    code: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    description: Mapped[str | None] = mapped_column(Text)
    created_by: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"))


class Project(UUIDPrimaryKeyMixin, TimestampMixin, Base):
    __tablename__ = "projects"
    __table_args__ = (
        UniqueConstraint("team_space_id", "code", name="uq_projects_team_space_code"),
        UniqueConstraint("team_space_id", "name", name="uq_projects_team_space_name"),
        Index("idx_projects_team_space_id", "team_space_id"),
        Index("idx_projects_owner_id", "owner_id"),
    )

    team_space_id: Mapped[str] = mapped_column(
        UUID(as_uuid=True), ForeignKey("team_spaces.id"), nullable=False
    )
    name: Mapped[str] = mapped_column(String(128), nullable=False)
    code: Mapped[str] = mapped_column(String(64), nullable=False)
    description: Mapped[str | None] = mapped_column(Text)
    owner_id: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    status: Mapped[str] = mapped_column(String(16), nullable=False, default="active")


class Folder(UUIDPrimaryKeyMixin, TimestampMixin, Base):
    __tablename__ = "folders"
    __table_args__ = (
        UniqueConstraint("project_id", "parent_id", "name", name="uq_folders_project_parent_name"),
        UniqueConstraint("project_id", "path", name="uq_folders_project_path"),
        Index("idx_folders_project_id", "project_id"),
        Index("idx_folders_parent_id", "parent_id"),
    )

    project_id: Mapped[str] = mapped_column(UUID(as_uuid=True), ForeignKey("projects.id"), nullable=False)
    parent_id: Mapped[str | None] = mapped_column(UUID(as_uuid=True), ForeignKey("folders.id"))
    name: Mapped[str] = mapped_column(String(128), nullable=False)
    path: Mapped[str] = mapped_column(String(512), nullable=False)
    sort_order: Mapped[int] = mapped_column(nullable=False, default=0)
