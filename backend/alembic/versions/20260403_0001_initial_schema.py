"""initial schema

Revision ID: 20260403_0001
Revises:
Create Date: 2026-04-03 19:20:00
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision: str = "20260403_0001"
down_revision: str | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


user_role = sa.Enum("member", "project_lead", "admin", name="user_role")
document_status = sa.Enum(
    "draft",
    "in_progress",
    "pending_handover",
    "handed_over",
    "finalized",
    "archived",
    name="document_status",
)
handover_status = sa.Enum(
    "generated",
    "pending_confirm",
    "completed",
    "cancelled",
    name="handover_status",
)
suggestion_status = sa.Enum(
    "pending", "confirmed", "dismissed", "expired", name="suggestion_status"
)
suggestion_type = sa.Enum(
    "document_summary",
    "document_tag",
    "risk_alert",
    "handover_summary",
    "archive_recommendation",
    "structure_recommendation",
    name="suggestion_type",
)
audit_action_type = sa.Enum(
    "create",
    "view",
    "upload",
    "download",
    "replace_version",
    "transfer",
    "receive_transfer",
    "finalize",
    "archive",
    "restore",
    "delete",
    "handover_generate",
    "handover_confirm",
    "handover_complete",
    "admin_update",
    "ai_generate",
    "ai_confirm",
    "ai_dismiss",
    name="audit_action_type",
)


def upgrade() -> None:
    bind = op.get_bind()
    user_role.create(bind, checkfirst=True)
    document_status.create(bind, checkfirst=True)
    handover_status.create(bind, checkfirst=True)
    suggestion_status.create(bind, checkfirst=True)
    suggestion_type.create(bind, checkfirst=True)
    audit_action_type.create(bind, checkfirst=True)

    op.create_table(
        "users",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("username", sa.String(length=64), nullable=False),
        sa.Column("password_hash", sa.String(length=255), nullable=False),
        sa.Column("display_name", sa.String(length=64), nullable=False),
        sa.Column("role", user_role, nullable=False),
        sa.Column("email", sa.String(length=128)),
        sa.Column("phone", sa.String(length=32)),
        sa.Column("status", sa.String(length=16), nullable=False, server_default="active"),
        sa.Column("last_login_at", sa.DateTime(timezone=True)),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.UniqueConstraint("username"),
    )
    op.create_index("idx_users_role", "users", ["role"])
    op.create_index("idx_users_status", "users", ["status"])

    op.create_table(
        "team_spaces",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("name", sa.String(length=128), nullable=False),
        sa.Column("code", sa.String(length=64), nullable=False),
        sa.Column("description", sa.Text()),
        sa.Column("created_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.UniqueConstraint("name"),
        sa.UniqueConstraint("code"),
    )

    op.create_table(
        "projects",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("team_space_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("team_spaces.id"), nullable=False),
        sa.Column("name", sa.String(length=128), nullable=False),
        sa.Column("code", sa.String(length=64), nullable=False),
        sa.Column("description", sa.Text()),
        sa.Column("owner_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("status", sa.String(length=16), nullable=False, server_default="active"),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.UniqueConstraint("team_space_id", "code", name="uq_projects_team_space_code"),
        sa.UniqueConstraint("team_space_id", "name", name="uq_projects_team_space_name"),
    )
    op.create_index("idx_projects_team_space_id", "projects", ["team_space_id"])
    op.create_index("idx_projects_owner_id", "projects", ["owner_id"])

    op.create_table(
        "folders",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("project_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("projects.id"), nullable=False),
        sa.Column("parent_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("folders.id")),
        sa.Column("name", sa.String(length=128), nullable=False),
        sa.Column("path", sa.String(length=512), nullable=False),
        sa.Column("sort_order", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.UniqueConstraint("project_id", "parent_id", "name", name="uq_folders_project_parent_name"),
        sa.UniqueConstraint("project_id", "path", name="uq_folders_project_path"),
    )
    op.create_index("idx_folders_project_id", "folders", ["project_id"])
    op.create_index("idx_folders_parent_id", "folders", ["parent_id"])

    op.create_table(
        "documents",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("team_space_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("team_spaces.id"), nullable=False),
        sa.Column("project_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("projects.id"), nullable=False),
        sa.Column("folder_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("folders.id")),
        sa.Column("title", sa.String(length=255), nullable=False),
        sa.Column("description", sa.Text()),
        sa.Column("file_type", sa.String(length=32)),
        sa.Column("current_owner_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("current_status", document_status, nullable=False),
        sa.Column("current_version_id", postgresql.UUID(as_uuid=True)),
        sa.Column("is_archived", sa.Boolean(), nullable=False, server_default=sa.false()),
        sa.Column("is_deleted", sa.Boolean(), nullable=False, server_default=sa.false()),
        sa.Column("created_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False, server_default=sa.func.now()),
        sa.Column("deleted_at", sa.DateTime(timezone=True)),
        sa.Column("deleted_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
    )
    op.create_index("idx_documents_project_id", "documents", ["project_id"])
    op.create_index("idx_documents_folder_id", "documents", ["folder_id"])
    op.create_index("idx_documents_owner_id", "documents", ["current_owner_id"])
    op.create_index("idx_documents_status", "documents", ["current_status"])
    op.create_index("idx_documents_project_status", "documents", ["project_id", "current_status"])
    op.create_index("idx_documents_updated_at", "documents", ["updated_at"])

    op.create_table(
        "document_versions",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("document_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("documents.id"), nullable=False),
        sa.Column("version_no", sa.Integer(), nullable=False),
        sa.Column("file_name", sa.String(length=255), nullable=False),
        sa.Column("mime_type", sa.String(length=128)),
        sa.Column("file_size", sa.BigInteger(), nullable=False),
        sa.Column("storage_provider", sa.String(length=32), nullable=False),
        sa.Column("storage_bucket_or_share", sa.String(length=255)),
        sa.Column("storage_object_key", sa.String(length=1024), nullable=False),
        sa.Column("external_file_id", sa.String(length=255)),
        sa.Column("external_path", sa.String(length=1024)),
        sa.Column("commit_message", sa.String(length=500)),
        sa.Column("extracted_text_status", sa.String(length=16), nullable=False, server_default="pending"),
        sa.Column("summary_status", sa.String(length=16), nullable=False, server_default="pending"),
        sa.Column("summary_text", sa.Text()),
        sa.Column("created_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("document_id", "version_no", name="uq_document_versions_document_version"),
    )
    op.create_index("idx_document_versions_document_id", "document_versions", ["document_id"])
    op.create_index("idx_document_versions_created_at", "document_versions", ["created_at"])
    op.create_index("idx_document_versions_summary_status", "document_versions", ["summary_status"])

    op.create_table(
        "flow_records",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("document_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("documents.id"), nullable=False),
        sa.Column("version_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("document_versions.id")),
        sa.Column("from_user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("to_user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("from_status", document_status),
        sa.Column("to_status", document_status, nullable=False),
        sa.Column("action", sa.String(length=32), nullable=False),
        sa.Column("note", sa.String(length=500)),
        sa.Column("created_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
    )
    op.create_index("idx_flow_records_document_id", "flow_records", ["document_id"])
    op.create_index("idx_flow_records_to_user_id", "flow_records", ["to_user_id"])
    op.create_index("idx_flow_records_created_at", "flow_records", ["created_at"])

    op.create_table(
        "graduation_handovers",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("target_user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("receiver_user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("project_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("projects.id")),
        sa.Column("status", handover_status, nullable=False),
        sa.Column("remark", sa.String(length=500)),
        sa.Column("ai_summary", sa.Text()),
        sa.Column("generated_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id"), nullable=False),
        sa.Column("generated_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("confirmed_at", sa.DateTime(timezone=True)),
        sa.Column("completed_at", sa.DateTime(timezone=True)),
        sa.Column("cancelled_at", sa.DateTime(timezone=True)),
    )
    op.create_index("idx_graduation_handovers_target_user_id", "graduation_handovers", ["target_user_id"])
    op.create_index(
        "idx_graduation_handovers_receiver_user_id", "graduation_handovers", ["receiver_user_id"]
    )
    op.create_index("idx_graduation_handovers_status", "graduation_handovers", ["status"])

    op.create_table(
        "graduation_handover_items",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("handover_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("graduation_handovers.id"), nullable=False),
        sa.Column("document_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("documents.id"), nullable=False),
        sa.Column("selected", sa.Boolean(), nullable=False, server_default=sa.true()),
        sa.Column("note", sa.String(length=500)),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("handover_id", "document_id", name="uq_handover_items_handover_document"),
    )

    op.create_table(
        "audit_events",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("document_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("documents.id")),
        sa.Column("version_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("document_versions.id")),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("action_type", audit_action_type, nullable=False),
        sa.Column("request_id", sa.String(length=64)),
        sa.Column("ip_address", postgresql.INET()),
        sa.Column("terminal_info", sa.String(length=255)),
        sa.Column("extra_data", postgresql.JSONB(astext_type=sa.Text())),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
    )
    op.create_index("idx_audit_events_document_id", "audit_events", ["document_id"])
    op.create_index("idx_audit_events_user_id", "audit_events", ["user_id"])
    op.create_index("idx_audit_events_action_type", "audit_events", ["action_type"])
    op.create_index("idx_audit_events_created_at", "audit_events", ["created_at"])

    op.create_table(
        "assistant_suggestions",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("related_type", sa.String(length=32), nullable=False),
        sa.Column("related_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("suggestion_type", suggestion_type, nullable=False),
        sa.Column("status", suggestion_status, nullable=False),
        sa.Column("title", sa.String(length=255)),
        sa.Column("content", sa.Text(), nullable=False),
        sa.Column("source_scope", sa.String(length=255)),
        sa.Column("confidence", sa.Numeric(5, 4)),
        sa.Column("request_id", sa.String(length=64)),
        sa.Column("generated_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("expires_at", sa.DateTime(timezone=True)),
        sa.Column("confirmed_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("confirmed_at", sa.DateTime(timezone=True)),
        sa.Column("dismissed_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("dismissed_at", sa.DateTime(timezone=True)),
    )
    op.create_index(
        "idx_assistant_suggestions_related",
        "assistant_suggestions",
        ["related_type", "related_id"],
    )
    op.create_index("idx_assistant_suggestions_status", "assistant_suggestions", ["status"])
    op.create_index("idx_assistant_suggestions_type", "assistant_suggestions", ["suggestion_type"])
    op.create_index("idx_assistant_suggestions_generated_at", "assistant_suggestions", ["generated_at"])

    op.create_table(
        "assistant_requests",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True, nullable=False),
        sa.Column("request_type", sa.String(length=32), nullable=False),
        sa.Column("related_type", sa.String(length=32)),
        sa.Column("related_id", postgresql.UUID(as_uuid=True)),
        sa.Column("payload", postgresql.JSONB(astext_type=sa.Text())),
        sa.Column("status", sa.String(length=16), nullable=False),
        sa.Column("error_message", sa.Text()),
        sa.Column("created_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id")),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("completed_at", sa.DateTime(timezone=True)),
    )


def downgrade() -> None:
    op.drop_table("assistant_requests")
    op.drop_index("idx_assistant_suggestions_generated_at", table_name="assistant_suggestions")
    op.drop_index("idx_assistant_suggestions_type", table_name="assistant_suggestions")
    op.drop_index("idx_assistant_suggestions_status", table_name="assistant_suggestions")
    op.drop_index("idx_assistant_suggestions_related", table_name="assistant_suggestions")
    op.drop_table("assistant_suggestions")
    op.drop_index("idx_audit_events_created_at", table_name="audit_events")
    op.drop_index("idx_audit_events_action_type", table_name="audit_events")
    op.drop_index("idx_audit_events_user_id", table_name="audit_events")
    op.drop_index("idx_audit_events_document_id", table_name="audit_events")
    op.drop_table("audit_events")
    op.drop_table("graduation_handover_items")
    op.drop_index("idx_graduation_handovers_status", table_name="graduation_handovers")
    op.drop_index("idx_graduation_handovers_receiver_user_id", table_name="graduation_handovers")
    op.drop_index("idx_graduation_handovers_target_user_id", table_name="graduation_handovers")
    op.drop_table("graduation_handovers")
    op.drop_index("idx_flow_records_created_at", table_name="flow_records")
    op.drop_index("idx_flow_records_to_user_id", table_name="flow_records")
    op.drop_index("idx_flow_records_document_id", table_name="flow_records")
    op.drop_table("flow_records")
    op.drop_index("idx_document_versions_summary_status", table_name="document_versions")
    op.drop_index("idx_document_versions_created_at", table_name="document_versions")
    op.drop_index("idx_document_versions_document_id", table_name="document_versions")
    op.drop_table("document_versions")
    op.drop_index("idx_documents_updated_at", table_name="documents")
    op.drop_index("idx_documents_project_status", table_name="documents")
    op.drop_index("idx_documents_status", table_name="documents")
    op.drop_index("idx_documents_owner_id", table_name="documents")
    op.drop_index("idx_documents_folder_id", table_name="documents")
    op.drop_index("idx_documents_project_id", table_name="documents")
    op.drop_table("documents")
    op.drop_index("idx_folders_parent_id", table_name="folders")
    op.drop_index("idx_folders_project_id", table_name="folders")
    op.drop_table("folders")
    op.drop_index("idx_projects_owner_id", table_name="projects")
    op.drop_index("idx_projects_team_space_id", table_name="projects")
    op.drop_table("projects")
    op.drop_table("team_spaces")
    op.drop_index("idx_users_status", table_name="users")
    op.drop_index("idx_users_role", table_name="users")
    op.drop_table("users")
    audit_action_type.drop(op.get_bind(), checkfirst=True)
    suggestion_type.drop(op.get_bind(), checkfirst=True)
    suggestion_status.drop(op.get_bind(), checkfirst=True)
    handover_status.drop(op.get_bind(), checkfirst=True)
    document_status.drop(op.get_bind(), checkfirst=True)
    user_role.drop(op.get_bind(), checkfirst=True)
