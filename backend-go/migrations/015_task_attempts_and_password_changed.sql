-- 015_task_attempts_and_password_changed.sql
-- 1. assistant_requests.attempts：任务被领取的次数。卡死恢复达到上限后标记
--    失败（死信），不再无限重新排队；避免一个总让 Worker 卡死的文件反复
--    占用 Worker 并重复调用模型。
-- 2. users.password_changed_at：改密时间。签发时间早于它的 JWT 一律拒绝，
--    实现「改密即登出其他会话」。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 15) THEN
        RAISE EXCEPTION 'migration 015 already applied';
    END IF;
END $$;

ALTER TABLE assistant_requests ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

INSERT INTO schema_migrations (version, name) VALUES (15, '015_task_attempts_and_password_changed');

COMMIT;
