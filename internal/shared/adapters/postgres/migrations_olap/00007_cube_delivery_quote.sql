-- +goose Up
CREATE TABLE IF NOT EXISTS analytics.cube_delivery_quote
(
    quote_id        UUID           PRIMARY KEY,
    customer_id     UUID           NOT NULL,
    shop_id         UUID           NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL,
    fee_usd         NUMERIC(12, 2) NOT NULL,
    fare_band       TEXT           NOT NULL,
    quote_date      DATE           NOT NULL,
    quote_hour      SMALLINT       NOT NULL,
    created_at      TIMESTAMPTZ    NOT NULL,
    synced_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cube_delivery_quote_created_at ON analytics.cube_delivery_quote (created_at);
CREATE INDEX IF NOT EXISTS idx_cube_delivery_quote_quote_date ON analytics.cube_delivery_quote (quote_date);
CREATE INDEX IF NOT EXISTS idx_cube_delivery_quote_fare_band ON analytics.cube_delivery_quote (fare_band);

-- El enlace de conversión: cube_order.quote_id apunta a la cotización que lo originó.
ALTER TABLE analytics.cube_order
    ADD COLUMN IF NOT EXISTS quote_id UUID;

CREATE INDEX IF NOT EXISTS idx_cube_order_quote_id ON analytics.cube_order (quote_id)
    WHERE quote_id IS NOT NULL;

-- +goose StatementBegin
-- Mismos cortes de distancia que la tarifa de trips: 4,5 km es donde termina
-- el piso de $1 y arranca el cobro por kilómetro (misma fórmula, tripfare).
CREATE OR REPLACE VIEW analytics.view_delivery_quote_conversion AS
WITH banded AS (
    SELECT
        q.quote_id,
        q.quote_date,
        q.quote_hour,
        q.fare_band,
        q.fee_usd,
        q.distance_meters,
        CASE
            WHEN q.distance_meters < 2000  THEN 0
            WHEN q.distance_meters < 4500  THEN 2
            WHEN q.distance_meters < 7000  THEN 4
            WHEN q.distance_meters < 10000 THEN 7
            WHEN q.distance_meters < 15000 THEN 10
            ELSE 15
        END AS distance_band_from_km
    FROM analytics.cube_delivery_quote q
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
    COUNT(o.order_id)                                          AS converted,
    ROUND(COUNT(o.order_id)::NUMERIC / COUNT(*), 4)            AS conversion_rate,
    ROUND(AVG(b.fee_usd), 2)                                   AS avg_fee_usd,
    ROUND(AVG(b.distance_meters))                              AS avg_distance_meters
FROM banded b
LEFT JOIN analytics.cube_order o ON o.quote_id = b.quote_id
GROUP BY b.quote_date, b.fare_band, b.distance_band_from_km;
-- +goose StatementEnd

-- +goose Down
DROP VIEW IF EXISTS analytics.view_delivery_quote_conversion;
DROP INDEX IF EXISTS analytics.idx_cube_order_quote_id;
ALTER TABLE analytics.cube_order DROP COLUMN IF EXISTS quote_id;
DROP TABLE IF EXISTS analytics.cube_delivery_quote;
