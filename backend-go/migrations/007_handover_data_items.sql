-- 007_handover_data_items.sql
-- 毕业交接单数据资产明细表

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version = 7) THEN
        RAISE EXCEPTION 'migration 007 already applied';
    END IF;
END $$;

CREATE TABLE graduation_handover_data_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    handover_id   UUID NOT NULL REFERENCES graduation_handovers(id),
    data_asset_id UUID NOT NULL REFERENCES data_assets(id),
    selected      BOOLEAN NOT NULL DEFAULT TRUE,
    note          VARCHAR(500),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_handover_data_item UNIQUE (handover_id, data_asset_id)
);

CREATE INDEX idx_handover_data_items_handover_id   ON graduation_handover_data_items (handover_id);
CREATE INDEX idx_handover_data_items_data_asset_id ON graduation_handover_data_items (data_asset_id);

INSERT INTO schema_migrations (version, name) VALUES (7, '007_handover_data_items');

COMMIT;
