import enum


class UserRole(str, enum.Enum):
    member = "member"
    project_lead = "project_lead"
    admin = "admin"


class DocumentStatus(str, enum.Enum):
    draft = "draft"
    in_progress = "in_progress"
    pending_handover = "pending_handover"
    handed_over = "handed_over"
    finalized = "finalized"
    archived = "archived"


class HandoverStatus(str, enum.Enum):
    generated = "generated"
    pending_confirm = "pending_confirm"
    completed = "completed"
    cancelled = "cancelled"


class SuggestionStatus(str, enum.Enum):
    pending = "pending"
    confirmed = "confirmed"
    dismissed = "dismissed"
    expired = "expired"


class SuggestionType(str, enum.Enum):
    document_summary = "document_summary"
    document_tag = "document_tag"
    risk_alert = "risk_alert"
    handover_summary = "handover_summary"
    archive_recommendation = "archive_recommendation"
    structure_recommendation = "structure_recommendation"


class AuditActionType(str, enum.Enum):
    create = "create"
    view = "view"
    upload = "upload"
    download = "download"
    replace_version = "replace_version"
    transfer = "transfer"
    receive_transfer = "receive_transfer"
    finalize = "finalize"
    archive = "archive"
    restore = "restore"
    delete = "delete"
    handover_generate = "handover_generate"
    handover_confirm = "handover_confirm"
    handover_complete = "handover_complete"
    admin_update = "admin_update"
    ai_generate = "ai_generate"
    ai_confirm = "ai_confirm"
    ai_dismiss = "ai_dismiss"

