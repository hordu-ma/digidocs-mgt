-- 016_chunk_embeddings.sql
-- 正文分块向量：bge-m3（1024 维），用于关键词 + 语义的混合检索。
-- 需要 pgvector 扩展（镜像 pgvector/pgvector:pg17-trixie）。扩展不可用时只输出
-- NOTICE、不建列，后端检测到无该列即退回纯关键词检索，不阻断启动。

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 16) THEN
        RAISE EXCEPTION 'migration 016 already applied';
    END IF;
END $$;

DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS vector;
    ALTER TABLE document_text_chunks ADD COLUMN IF NOT EXISTS embedding vector(1024);
    ALTER TABLE document_text_chunks ADD COLUMN IF NOT EXISTS embedding_model VARCHAR(64);
    CREATE INDEX IF NOT EXISTS idx_document_text_chunks_embedding_hnsw
        ON document_text_chunks USING hnsw (embedding vector_cosine_ops);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pgvector unavailable, semantic search disabled: %', SQLERRM;
END $$;

INSERT INTO schema_migrations (version, name) VALUES (16, '016_chunk_embeddings');

COMMIT;
