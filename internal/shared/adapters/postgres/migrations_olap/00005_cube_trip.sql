-- +goose Up
-- pickup_h3_cell/dropoff_h3_cell son resolución 7 (zona), copiados de trips.trips.pickup_h3_cell_zone/dropoff_h3_cell_zone — no confundir con trips.trips.pickup_h3_cell, que es resolución 8 (matching).
CREATE TABLE IF NOT EXISTS analytics.cube_trip
(
    trip_id         UUID           PRIMARY KEY,
    customer_id     UUID           NOT NULL,
    rider_id        UUID,
    status          TEXT           NOT NULL,
    fare            NUMERIC(12, 2),
    pickup_h3_cell  TEXT,
    dropoff_h3_cell TEXT,
    trip_date       DATE           NOT NULL,
    created_at      TIMESTAMPTZ    NOT NULL,
    synced_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cube_trip_created_at ON analytics.cube_trip (created_at);
CREATE INDEX IF NOT EXISTS idx_cube_trip_trip_date ON analytics.cube_trip (trip_date);
CREATE INDEX IF NOT EXISTS idx_cube_trip_status ON analytics.cube_trip (status);
CREATE INDEX IF NOT EXISTS idx_cube_trip_customer_id ON analytics.cube_trip (customer_id);
CREATE INDEX IF NOT EXISTS idx_cube_trip_rider_id ON analytics.cube_trip (rider_id)
    WHERE rider_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cube_trip_pickup_h3_cell ON analytics.cube_trip (pickup_h3_cell)
    WHERE pickup_h3_cell IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cube_trip_dropoff_h3_cell ON analytics.cube_trip (dropoff_h3_cell)
    WHERE dropoff_h3_cell IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS analytics.cube_trip;
