BEGIN;

-- Migration 016 could be recorded even if the server did not have pgvector.
-- Replay its DDL on the supported PostgreSQL image and fail visibly on error.
CREATE EXTENSION IF NOT EXISTS vector;
ALTER TABLE document_text_chunks ADD COLUMN IF NOT EXISTS embedding vector(1024);
ALTER TABLE document_text_chunks ADD COLUMN IF NOT EXISTS embedding_model varchar(64);
CREATE INDEX IF NOT EXISTS idx_document_text_chunks_embedding_hnsw
    ON document_text_chunks USING hnsw (embedding vector_cosine_ops);

INSERT INTO schema_migrations (version, name) VALUES (18, '018_verify_vector_schema');
COMMIT;
