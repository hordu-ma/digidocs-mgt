BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 2) THEN
        RAISE EXCEPTION 'migration 002 already applied';
    END IF;
END $$;

ALTER TABLE assistant_requests
    ADD COLUMN IF NOT EXISTS output JSONB;

INSERT INTO schema_migrations (version, name) VALUES (2, '002_assistant_request_output');

COMMIT;
