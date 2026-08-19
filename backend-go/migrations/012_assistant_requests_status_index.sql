-- 012_assistant_requests_status_index.sql
-- assistant_requests 从建表起就没有任何索引，但 Worker 轮询（约每 5 秒一次）
-- 固定跑 `WHERE status='running' AND started_at < ...` 的卡死恢复语句和
-- `WHERE status='pending' ORDER BY created_at ASC FOR UPDATE SKIP LOCKED`
-- 的领取语句。该表只增不减，量大后这两条语句会退化为全表顺序扫描并放大
-- 行锁范围。补一个 (status, created_at) 复合索引，覆盖两条语句的过滤 + 排序。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 12) THEN
        RAISE EXCEPTION 'migration 012 already applied';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_assistant_requests_status_created_at
    ON assistant_requests (status, created_at);

INSERT INTO schema_migrations (version, name) VALUES (12, '012_assistant_requests_status_index');

COMMIT;
