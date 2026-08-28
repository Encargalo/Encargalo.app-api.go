-- +goose Up
CREATE SCHEMA IF NOT EXISTS trips;

CREATE TABLE IF NOT EXISTS trips.trips
(
    id                 UUID             DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    customer_id        UUID             NOT NULL REFERENCES customers.accounts (id),
    rider_id           UUID,
    status             VARCHAR          NOT NULL DEFAULT 'requested',
    pickup_address     VARCHAR          NOT NULL,
    pickup_latitude    DOUBLE PRECISION NOT NULL,
    pickup_longitude   DOUBLE PRECISION NOT NULL,
    pickup_h3_cell     VARCHAR,
    dropoff_address    VARCHAR          NOT NULL,
    dropoff_latitude   DOUBLE PRECISION NOT NULL,
    dropoff_longitude  DOUBLE PRECISION NOT NULL,
    fare               NUMERIC(12, 2),
    requested_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    accepted_at        TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    cancelled_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trips_customer_id ON trips.trips (customer_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips.trips (status);
CREATE INDEX IF NOT EXISTS idx_trips_rider_id ON trips.trips (rider_id)
    WHERE rider_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trips_pickup_h3_cell ON trips.trips (pickup_h3_cell)
    WHERE pickup_h3_cell IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS trips.trips;
DROP SCHEMA IF EXISTS trips;
