-- +goose Up
CREATE SCHEMA IF NOT EXISTS tracking;

CREATE TABLE IF NOT EXISTS tracking.events
(
    id         UUID PRIMARY KEY,
    type       TEXT        NOT NULL,
    params     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tracking_events_created_at
    ON tracking.events (created_at);

CREATE INDEX IF NOT EXISTS idx_tracking_events_type_created_at
    ON tracking.events (type, created_at);

CREATE INDEX IF NOT EXISTS idx_tracking_events_params
    ON tracking.events USING GIN (params);

-- +goose Down
DROP TABLE IF EXISTS tracking.events;
DROP SCHEMA IF EXISTS tracking;
