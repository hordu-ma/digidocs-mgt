-- 006_data_assets.sql
-- 数据资产模块：data_folders + data_assets 表

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 6) THEN
        RAISE EXCEPTION 'migration 006 already applied';
    END IF;
END $$;

-- ========== data_folders ==========

CREATE TABLE data_folders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id),
    parent_id   UUID REFERENCES data_folders(id),
    depth       INTEGER NOT NULL DEFAULT 0,
    name        VARCHAR(128) NOT NULL,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_data_folder_parent_name UNIQUE (project_id, parent_id, name),
    CONSTRAINT chk_data_folder_depth CHECK (depth >= 0 AND depth <= 2)
);

CREATE INDEX idx_data_folders_project_id ON data_folders (project_id);
CREATE INDEX idx_data_folders_parent_id  ON data_folders (parent_id);

-- ========== data_assets ==========

CREATE TABLE data_assets (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_space_id           UUID NOT NULL REFERENCES team_spaces(id),
    project_id              UUID NOT NULL REFERENCES projects(id),
    folder_id               UUID REFERENCES data_folders(id),
    display_name            VARCHAR(255) NOT NULL,
    file_name               VARCHAR(255) NOT NULL,
    description             TEXT,
    mime_type               VARCHAR(128),
    file_size               BIGINT NOT NULL DEFAULT 0,
    storage_provider        VARCHAR(32) NOT NULL,
    storage_bucket_or_share VARCHAR(255),
    storage_object_key      VARCHAR(1024) NOT NULL,
    external_file_id        VARCHAR(255),
    external_path           VARCHAR(1024),
    created_by              UUID NOT NULL REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted              BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at              TIMESTAMPTZ,
    deleted_by              UUID REFERENCES users(id)
);

CREATE INDEX idx_data_assets_project_id  ON data_assets (project_id);
CREATE INDEX idx_data_assets_folder_id   ON data_assets (folder_id);
CREATE INDEX idx_data_assets_created_by  ON data_assets (created_by);
CREATE INDEX idx_data_assets_created_at  ON data_assets (created_at DESC);

INSERT INTO schema_migrations (version, name) VALUES (6, '006_data_assets');

COMMIT;
