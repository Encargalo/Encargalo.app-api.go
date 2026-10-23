-- +goose Up
-- fare queda como lo que paga el cliente; el rider cobra sobre fare + discount_usd.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS discount_usd NUMERIC(12, 2) NOT NULL DEFAULT 0;

ALTER TABLE trips.fare_quotes
    ADD COLUMN IF NOT EXISTS discount_usd NUMERIC(12, 2) NOT NULL DEFAULT 0;

-- Un solo viaje con descuento vivo o completado por cliente; cancelar o quedarse sin rider lo libera.
CREATE UNIQUE INDEX IF NOT EXISTS uq_trips_first_trip_discount ON trips.trips (customer_id)
    WHERE discount_usd > 0 AND status NOT IN ('cancelled', 'no_driver');

-- +goose Down
DROP INDEX IF EXISTS trips.uq_trips_first_trip_discount;
ALTER TABLE trips.fare_quotes DROP COLUMN IF EXISTS discount_usd;
ALTER TABLE trips.trips DROP COLUMN IF EXISTS discount_usd;
