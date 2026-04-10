BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 3) THEN
        RAISE EXCEPTION 'migration 003 already applied';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS assistant_conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type      VARCHAR(32) NOT NULL,
    scope_id        UUID NOT NULL,
    source_scope    JSONB NOT NULL DEFAULT '{}'::jsonb,
    title           VARCHAR(255),
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_assistant_conversations_scope ON assistant_conversations (scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_assistant_conversations_last_message_at ON assistant_conversations (last_message_at DESC);

CREATE TABLE IF NOT EXISTS assistant_conversation_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES assistant_conversations(id) ON DELETE CASCADE,
    role            VARCHAR(16) NOT NULL,
    content         TEXT NOT NULL,
    request_id      UUID,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assistant_conversation_messages_conversation ON assistant_conversation_messages (conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_assistant_conversation_messages_request_id ON assistant_conversation_messages (request_id);

ALTER TABLE assistant_requests
    ADD COLUMN IF NOT EXISTS conversation_id UUID REFERENCES assistant_conversations(id);

CREATE INDEX IF NOT EXISTS idx_assistant_requests_conversation_id ON assistant_requests (conversation_id);

INSERT INTO schema_migrations (version, name) VALUES (3, '003_assistant_conversations');

COMMIT;
