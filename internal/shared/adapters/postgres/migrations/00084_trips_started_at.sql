-- +goose Up
-- Momento en que el viaje pasó a in_progress. Los anteriores se rellenan con el uso del código de recogida.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;

UPDATE trips.trips
SET started_at = pickup_code_used_at
WHERE started_at IS NULL
  AND pickup_code_used_at IS NOT NULL;

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS started_at;
