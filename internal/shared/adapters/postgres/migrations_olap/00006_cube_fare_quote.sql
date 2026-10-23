-- +goose Up
CREATE TABLE IF NOT EXISTS analytics.cube_fare_quote
(
    quote_id        UUID           PRIMARY KEY,
    customer_id     UUID           NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL,
    duration_min    BIGINT         NOT NULL,
    fare_usd        NUMERIC(12, 2) NOT NULL,
    fare_band       TEXT           NOT NULL,
    pickup_h3_cell  TEXT,
    dropoff_h3_cell TEXT,
    quote_date      DATE           NOT NULL,
    quote_hour      SMALLINT       NOT NULL,
    created_at      TIMESTAMPTZ    NOT NULL,
    synced_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cube_fare_quote_created_at ON analytics.cube_fare_quote (created_at);
CREATE INDEX IF NOT EXISTS idx_cube_fare_quote_quote_date ON analytics.cube_fare_quote (quote_date);
CREATE INDEX IF NOT EXISTS idx_cube_fare_quote_fare_band ON analytics.cube_fare_quote (fare_band);

-- El enlace de conversión: cube_trip.quote_id apunta a la cotización que lo originó.
ALTER TABLE analytics.cube_trip
    ADD COLUMN IF NOT EXISTS quote_id UUID;

CREATE INDEX IF NOT EXISTS idx_cube_trip_quote_id ON analytics.cube_trip (quote_id)
    WHERE quote_id IS NOT NULL;

-- +goose StatementBegin
-- Los cortes de distancia siguen la estructura de la tarifa: 4,5 km es donde
-- termina el piso de $1 y arranca el cobro por kilómetro.
CREATE OR REPLACE VIEW analytics.view_fare_quote_conversion AS
WITH banded AS (
    SELECT
        q.quote_id,
        q.quote_date,
        q.quote_hour,
        q.fare_band,
        q.fare_usd,
        q.distance_meters,
        CASE
            WHEN q.distance_meters < 2000  THEN 0
            WHEN q.distance_meters < 4500  THEN 2
            WHEN q.distance_meters < 7000  THEN 4
            WHEN q.distance_meters < 10000 THEN 7
            WHEN q.distance_meters < 15000 THEN 10
            ELSE 15
        END AS distance_band_from_km
    FROM analytics.cube_fare_quote q
)
SELECT
    b.quote_date,
    b.fare_band,
    b.distance_band_from_km,
    CASE b.distance_band_from_km
        WHEN 0  THEN '0-2 km'
        WHEN 2  THEN '2-4.5 km'
        WHEN 4  THEN '4.5-7 km'
        WHEN 7  THEN '7-10 km'
        WHEN 10 THEN '10-15 km'
        ELSE '15+ km'
    END AS distance_band,
    COUNT(*)                                                   AS quotes,
    COUNT(t.trip_id)                                           AS converted,
    ROUND(COUNT(t.trip_id)::NUMERIC / COUNT(*), 4)             AS conversion_rate,
    ROUND(AVG(b.fare_usd), 2)                                  AS avg_fare_usd,
    ROUND(AVG(b.distance_meters))                              AS avg_distance_meters
FROM banded b
LEFT JOIN analytics.cube_trip t ON t.quote_id = b.quote_id
GROUP BY b.quote_date, b.fare_band, b.distance_band_from_km;
-- +goose StatementEnd

-- +goose Down
DROP VIEW IF EXISTS analytics.view_fare_quote_conversion;
DROP INDEX IF EXISTS analytics.idx_cube_trip_quote_id;
ALTER TABLE analytics.cube_trip DROP COLUMN IF EXISTS quote_id;
DROP TABLE IF EXISTS analytics.cube_fare_quote;
