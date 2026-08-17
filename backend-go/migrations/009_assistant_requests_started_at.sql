-- 009_assistant_requests_started_at.sql
-- 修复任务卡死恢复误判：原逻辑用 created_at 判定 running 是否卡死，
-- 会把刚被领走、正常处理中但排队较久的任务误判为卡死并重新入队，
-- 导致同一任务被并发领取、重复处理。新增 started_at 记录真正进入
-- running 的时间，恢复逻辑改为基于该字段判断。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 9) THEN
        RAISE EXCEPTION 'migration 009 already applied';
    END IF;
END $$;

ALTER TABLE assistant_requests ADD COLUMN started_at TIMESTAMPTZ;

INSERT INTO schema_migrations (version, name) VALUES (9, '009_assistant_requests_started_at');

COMMIT;
