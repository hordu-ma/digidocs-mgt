-- 011_project_invites.sql
-- 课题邀请链接：确认加入后只写入 project_members，不新增用户/文档状态。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 11) THEN
        RAISE EXCEPTION 'migration 011 already applied';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS project_invites (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id),
    token        VARCHAR(64) UNIQUE NOT NULL,
    project_role VARCHAR(32) NOT NULL DEFAULT 'contributor',
    status       VARCHAR(16) NOT NULL DEFAULT 'active',
    expires_at   TIMESTAMPTZ NOT NULL,
    created_by   UUID REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at   TIMESTAMPTZ,
    revoked_by   UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_project_invites_project_id ON project_invites (project_id);
CREATE INDEX IF NOT EXISTS idx_project_invites_token ON project_invites (token);
CREATE INDEX IF NOT EXISTS idx_project_invites_status ON project_invites (status);

INSERT INTO schema_migrations (version, name) VALUES (11, '011_project_invites');

COMMIT;
