-- 001_initial_schema.sql
-- DigiDocs Mgt 初始数据库 schema
-- 基于 docs/数据库设计.md 定义

BEGIN;

-- ========== 枚举类型 ==========

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('member', 'project_lead', 'admin');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE document_status AS ENUM (
        'draft', 'in_progress', 'pending_handover', 'handed_over', 'finalized', 'archived'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE handover_status AS ENUM (
        'generated', 'pending_confirm', 'completed', 'cancelled'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ========== 迁移版本跟踪表 ==========

CREATE TABLE IF NOT EXISTS schema_migrations (
    version  INTEGER PRIMARY KEY,
    name     VARCHAR(255) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 幂等检查：如果已经执行过则跳过
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 1) THEN
        RAISE EXCEPTION 'migration 001 already applied';
    END IF;
END $$;

-- ========== users ==========

CREATE TABLE IF NOT EXISTS users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username       VARCHAR(64) UNIQUE NOT NULL,
    password_hash  VARCHAR(255) NOT NULL,
    display_name   VARCHAR(64) NOT NULL,
    role           user_role NOT NULL DEFAULT 'member',
    email          VARCHAR(128),
    phone          VARCHAR(32),
    status         VARCHAR(16) NOT NULL DEFAULT 'active',
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);

-- ========== team_spaces ==========

CREATE TABLE IF NOT EXISTS team_spaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) UNIQUE NOT NULL,
    code        VARCHAR(64) UNIQUE NOT NULL,
    description TEXT,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========== projects ==========

CREATE TABLE IF NOT EXISTS projects (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_space_id UUID NOT NULL REFERENCES team_spaces(id),
    name          VARCHAR(128) NOT NULL,
    code          VARCHAR(64) NOT NULL,
    description   TEXT,
    owner_id      UUID NOT NULL REFERENCES users(id),
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_projects_team_space_code UNIQUE (team_space_id, code),
    CONSTRAINT uq_projects_team_space_name UNIQUE (team_space_id, name)
);

CREATE INDEX IF NOT EXISTS idx_projects_team_space_id ON projects (team_space_id);
CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects (owner_id);

-- ========== folders ==========

CREATE TABLE IF NOT EXISTS folders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id),
    parent_id  UUID REFERENCES folders(id),
    name       VARCHAR(128) NOT NULL,
    path       VARCHAR(512) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_folders_project_parent_name UNIQUE (project_id, parent_id, name),
    CONSTRAINT uq_folders_project_path UNIQUE (project_id, path)
);

CREATE INDEX IF NOT EXISTS idx_folders_project_id ON folders (project_id);
CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders (parent_id);

-- ========== documents ==========

CREATE TABLE IF NOT EXISTS documents (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_space_id      UUID NOT NULL REFERENCES team_spaces(id),
    project_id         UUID NOT NULL REFERENCES projects(id),
    folder_id          UUID REFERENCES folders(id),
    title              VARCHAR(255) NOT NULL,
    description        TEXT,
    file_type          VARCHAR(32),
    current_owner_id   UUID NOT NULL REFERENCES users(id),
    current_status     document_status NOT NULL DEFAULT 'draft',
    current_version_id UUID,
    is_archived        BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted         BOOLEAN NOT NULL DEFAULT FALSE,
    created_by         UUID NOT NULL REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ,
    deleted_by         UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_documents_project_id ON documents (project_id);
CREATE INDEX IF NOT EXISTS idx_documents_folder_id ON documents (folder_id);
CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON documents (current_owner_id);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents (current_status);
CREATE INDEX IF NOT EXISTS idx_documents_project_status ON documents (project_id, current_status);
CREATE INDEX IF NOT EXISTS idx_documents_updated_at ON documents (updated_at);

-- ========== document_versions ==========

CREATE TABLE IF NOT EXISTS document_versions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id            UUID NOT NULL REFERENCES documents(id),
    version_no             INTEGER NOT NULL,
    file_name              VARCHAR(255) NOT NULL,
    mime_type              VARCHAR(128),
    file_size              BIGINT NOT NULL,
    storage_provider       VARCHAR(32) NOT NULL,
    storage_bucket_or_share VARCHAR(255),
    storage_object_key     VARCHAR(1024) NOT NULL,
    external_file_id       VARCHAR(255),
    external_path          VARCHAR(1024),
    commit_message         VARCHAR(500),
    extracted_text_status  VARCHAR(16) NOT NULL DEFAULT 'pending',
    summary_status         VARCHAR(16) NOT NULL DEFAULT 'pending',
    summary_text           TEXT,
    created_by             UUID NOT NULL REFERENCES users(id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_document_versions_doc_no UNIQUE (document_id, version_no)
);

CREATE INDEX IF NOT EXISTS idx_document_versions_document_id ON document_versions (document_id);
CREATE INDEX IF NOT EXISTS idx_document_versions_created_at ON document_versions (created_at);
CREATE INDEX IF NOT EXISTS idx_document_versions_summary_status ON document_versions (summary_status);

-- ========== flow_records ==========

CREATE TABLE IF NOT EXISTS flow_records (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id   UUID NOT NULL REFERENCES documents(id),
    version_id    UUID REFERENCES document_versions(id),
    from_user_id  UUID REFERENCES users(id),
    to_user_id    UUID REFERENCES users(id),
    from_status   document_status,
    to_status     document_status NOT NULL,
    action        VARCHAR(32) NOT NULL,
    note          VARCHAR(500),
    created_by    UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_flow_records_document_id ON flow_records (document_id);
CREATE INDEX IF NOT EXISTS idx_flow_records_to_user_id ON flow_records (to_user_id);
CREATE INDEX IF NOT EXISTS idx_flow_records_created_at ON flow_records (created_at);

-- ========== graduation_handovers ==========

CREATE TABLE IF NOT EXISTS graduation_handovers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_user_id   UUID NOT NULL REFERENCES users(id),
    receiver_user_id UUID NOT NULL REFERENCES users(id),
    project_id       UUID REFERENCES projects(id),
    status           handover_status NOT NULL DEFAULT 'generated',
    remark           VARCHAR(500),
    ai_summary       TEXT,
    generated_by     UUID NOT NULL REFERENCES users(id),
    generated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at     TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    cancelled_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_graduation_handovers_target ON graduation_handovers (target_user_id);
CREATE INDEX IF NOT EXISTS idx_graduation_handovers_receiver ON graduation_handovers (receiver_user_id);
CREATE INDEX IF NOT EXISTS idx_graduation_handovers_status ON graduation_handovers (status);

-- ========== graduation_handover_items ==========

CREATE TABLE IF NOT EXISTS graduation_handover_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    handover_id  UUID NOT NULL REFERENCES graduation_handovers(id),
    document_id  UUID NOT NULL REFERENCES documents(id),
    selected     BOOLEAN NOT NULL DEFAULT TRUE,
    note         VARCHAR(500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_handover_items_handover_doc UNIQUE (handover_id, document_id)
);

-- ========== audit_events ==========

CREATE TABLE IF NOT EXISTS audit_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id    UUID REFERENCES documents(id),
    version_id     UUID REFERENCES document_versions(id),
    user_id        UUID REFERENCES users(id),
    action_type    VARCHAR(32) NOT NULL,
    request_id     VARCHAR(64),
    ip_address     INET,
    terminal_info  VARCHAR(255),
    extra_data     JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_document_id ON audit_events (document_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_action_type ON audit_events (action_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events (created_at);

-- ========== assistant_suggestions ==========

CREATE TABLE IF NOT EXISTS assistant_suggestions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    related_type    VARCHAR(32) NOT NULL,
    related_id      UUID NOT NULL,
    suggestion_type VARCHAR(32) NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    title           VARCHAR(255),
    content         TEXT NOT NULL,
    source_scope    VARCHAR(255),
    confidence      NUMERIC(5,4),
    request_id      VARCHAR(64),
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    confirmed_by    UUID REFERENCES users(id),
    confirmed_at    TIMESTAMPTZ,
    dismissed_by    UUID REFERENCES users(id),
    dismissed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_assistant_suggestions_related ON assistant_suggestions (related_type, related_id);
CREATE INDEX IF NOT EXISTS idx_assistant_suggestions_status ON assistant_suggestions (status);
CREATE INDEX IF NOT EXISTS idx_assistant_suggestions_type ON assistant_suggestions (suggestion_type);
CREATE INDEX IF NOT EXISTS idx_assistant_suggestions_generated_at ON assistant_suggestions (generated_at);

-- ========== assistant_requests ==========

CREATE TABLE IF NOT EXISTS assistant_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_type  VARCHAR(32) NOT NULL,
    related_type  VARCHAR(32),
    related_id    UUID,
    payload       JSONB,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_by    UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

-- ========== 记录迁移版本 ==========

INSERT INTO schema_migrations (version, name) VALUES (1, '001_initial_schema');

COMMIT;
