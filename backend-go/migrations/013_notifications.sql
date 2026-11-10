-- 013_notifications.sql
-- 站内通知：转交、交接、AI 摘要完成、入组等事件的附属提醒。
-- 通知只是提醒，不是业务事实；业务状态仍以 documents / flow_records /
-- graduation_handovers 为准。待办收件箱由业务表实时计算，不落库。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 13) THEN
        RAISE EXCEPTION 'migration 013 already applied';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS notifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id),
    event_type    VARCHAR(48) NOT NULL,
    title         VARCHAR(255) NOT NULL,
    body          TEXT,
    link          VARCHAR(255),
    related_type  VARCHAR(32),
    related_id    UUID,
    actor_id      UUID REFERENCES users(id),
    read_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 未读计数与列表都按 (user_id, created_at) 取，未读角标再按 read_at IS NULL 过滤。
CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications (user_id) WHERE read_at IS NULL;

INSERT INTO schema_migrations (version, name) VALUES (13, '013_notifications');

COMMIT;
