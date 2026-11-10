-- 014_document_text_chunks.sql
-- 1. 文档正文分块：AI 检索问答与全文检索共用。此前正文只存在
--    assistant_requests.output 的 JSON 里，无法检索。
-- 2. assistant_suggestions.action_payload：AI 建议可附带一个白名单内的
--    流转动作，人工「确认并执行」时才由主后端以确认人身份执行。
-- 正文与建议仍是附属数据，不改变任何业务状态。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 14) THEN
        RAISE EXCEPTION 'migration 014 already applied';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS document_text_chunks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID NOT NULL REFERENCES documents(id),
    version_id   UUID NOT NULL REFERENCES document_versions(id),
    chunk_no     INTEGER NOT NULL,
    content      TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_document_text_chunks_version_chunk UNIQUE (version_id, chunk_no)
);

CREATE INDEX IF NOT EXISTS idx_document_text_chunks_document_id ON document_text_chunks (document_id);

ALTER TABLE assistant_suggestions ADD COLUMN IF NOT EXISTS action_payload JSONB;

-- Trigram index speeds up ILIKE for terms of 3+ characters. pg_trgm is a
-- trusted extension, but a restricted database role may still be unable to
-- create it; search stays correct without the index (sequential scan), so a
-- failure here must not block the migration and with it the whole app.
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
    CREATE INDEX IF NOT EXISTS idx_document_text_chunks_content_trgm
        ON document_text_chunks USING gin (content gin_trgm_ops);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_trgm unavailable, text search runs without index: %', SQLERRM;
END $$;

-- Backfill: the newest completed extraction of each document's *current*
-- version, chunked the same way as the application (800 chars, 100 overlap).
WITH latest AS (
    SELECT DISTINCT ON (COALESCE(NULLIF(r.output->>'version_id', ''), r.payload->>'version_id'))
        d.id AS document_id,
        d.current_version_id AS version_id,
        r.output->>'extracted_text' AS body
    FROM assistant_requests r
    JOIN documents d
      ON d.current_version_id::text = COALESCE(NULLIF(r.output->>'version_id', ''), r.payload->>'version_id')
    WHERE r.status = 'completed'
      AND r.request_type IN ('document.extract_text', 'document.summarize')
      AND length(COALESCE(r.output->>'extracted_text', '')) > 0
    ORDER BY COALESCE(NULLIF(r.output->>'version_id', ''), r.payload->>'version_id'), r.completed_at DESC NULLS LAST
)
INSERT INTO document_text_chunks (document_id, version_id, chunk_no, content)
SELECT l.document_id, l.version_id, g.n, substring(l.body FROM g.n * 700 + 1 FOR 800)
FROM latest l
CROSS JOIN LATERAL generate_series(0, GREATEST((length(l.body) - 101) / 700, 0)) AS g(n)
WHERE g.n < 400
ON CONFLICT (version_id, chunk_no) DO NOTHING;

INSERT INTO schema_migrations (version, name) VALUES (14, '014_document_text_chunks');

COMMIT;
