-- +goose Up
CREATE TABLE IF NOT EXISTS analytics.fact_tracking_event
(
    id         UUID PRIMARY KEY,
    type       TEXT        NOT NULL,
    params     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    synced_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fact_tracking_event_created_at
    ON analytics.fact_tracking_event (created_at);

CREATE INDEX IF NOT EXISTS idx_fact_tracking_event_type_created_at
    ON analytics.fact_tracking_event (type, created_at);

CREATE INDEX IF NOT EXISTS idx_fact_tracking_event_params
    ON analytics.fact_tracking_event USING GIN (params);

-- +goose Down
DROP TABLE IF EXISTS analytics.fact_tracking_event;
