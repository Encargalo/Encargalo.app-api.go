-- +goose Up
-- Código estable que la app traduce; nunca texto del staff ni quién canceló.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS cancel_reason TEXT;

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS cancel_reason;
