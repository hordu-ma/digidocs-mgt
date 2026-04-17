-- 005_project_members_permissions.sql
-- Add project-scoped membership used by the first permission matrix.

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 5) THEN
        RAISE EXCEPTION 'migration 005 already applied';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS project_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id),
    user_id      UUID NOT NULL REFERENCES users(id),
    project_role VARCHAR(32) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_project_members_project_user UNIQUE (project_id, user_id),
    CONSTRAINT ck_project_members_role CHECK (project_role IN ('owner', 'manager', 'contributor', 'viewer'))
);

CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members (project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members (user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_project_role ON project_members (project_role);

INSERT INTO project_members (project_id, user_id, project_role)
SELECT id, owner_id, 'owner'
FROM projects
ON CONFLICT ON CONSTRAINT uq_project_members_project_user DO UPDATE
SET project_role = EXCLUDED.project_role,
    updated_at = NOW();

INSERT INTO schema_migrations (version, name) VALUES (5, '005_project_members_permissions');

COMMIT;
