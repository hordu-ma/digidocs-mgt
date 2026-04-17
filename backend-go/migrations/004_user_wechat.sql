-- 004_user_wechat.sql
-- Add optional WeChat contact field to platform users.

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 4) THEN
        RAISE EXCEPTION 'migration 004 already applied';
    END IF;
END $$;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS wechat VARCHAR(64);

INSERT INTO schema_migrations (version, name) VALUES (4, '004_user_wechat');

COMMIT;
