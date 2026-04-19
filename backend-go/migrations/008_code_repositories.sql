-- 008_code_repositories.sql
-- 代码模块：受控 Git 仓库配置与 push 同步事件

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 8) THEN
        RAISE EXCEPTION 'migration 008 already applied';
    END IF;
END $$;

CREATE TABLE code_repositories (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_space_id      UUID NOT NULL REFERENCES team_spaces(id),
    project_id         UUID NOT NULL REFERENCES projects(id),
    name               VARCHAR(128) NOT NULL,
    slug               VARCHAR(96) NOT NULL UNIQUE,
    description        TEXT,
    default_branch     VARCHAR(128) NOT NULL DEFAULT 'main',
    target_folder_path VARCHAR(1024) NOT NULL,
    repo_storage_path  VARCHAR(1024) NOT NULL,
    push_token         VARCHAR(128) NOT NULL,
    last_commit_sha    VARCHAR(64),
    last_pushed_at     TIMESTAMPTZ,
    status             VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by         UUID NOT NULL REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted         BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at         TIMESTAMPTZ,
    deleted_by         UUID REFERENCES users(id),
    CONSTRAINT uq_code_repositories_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_code_repositories_project_id ON code_repositories(project_id);
CREATE INDEX idx_code_repositories_created_by ON code_repositories(created_by);
CREATE INDEX idx_code_repositories_status ON code_repositories(status);

CREATE TABLE code_push_events (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_repository_id UUID NOT NULL REFERENCES code_repositories(id),
    branch             VARCHAR(128) NOT NULL,
    before_sha         VARCHAR(64),
    after_sha          VARCHAR(64),
    commit_message     VARCHAR(500),
    pusher_id          UUID REFERENCES users(id),
    sync_status        VARCHAR(32) NOT NULL,
    error_message      TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at       TIMESTAMPTZ
);

CREATE INDEX idx_code_push_events_repo_id ON code_push_events(code_repository_id);
CREATE INDEX idx_code_push_events_created_at ON code_push_events(created_at DESC);
CREATE INDEX idx_code_push_events_status ON code_push_events(sync_status);

INSERT INTO schema_migrations (version, name) VALUES (8, '008_code_repositories');

COMMIT;
