-- 010_self_registration.sql
-- 开放自助注册：username 改为可空，email / phone 在非空时唯一。
-- 老账号用户名登录不受影响；自助注册账号不设 username。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 10) THEN
        RAISE EXCEPTION 'migration 010 already applied';
    END IF;
END $$;

ALTER TABLE users ALTER COLUMN username DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users (phone) WHERE phone IS NOT NULL;

INSERT INTO schema_migrations (version, name) VALUES (10, '010_self_registration');

COMMIT;
