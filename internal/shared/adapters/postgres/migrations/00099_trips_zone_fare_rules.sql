-- +goose Up
-- zone_fare_rules/zone_fare_cells permiten que una zona geográfica (ej.
-- Charallave) tenga su propia mínima y tarifa/km por banda horaria, sin
-- tocar código ni desplegar (ver issue #376). zone_fare_cells normaliza la
-- relación celda→zona en vez de un array, porque la consulta caliente de
-- tripsApp.EstimateFare es "¿qué regla tiene esta celda puntual?": con
-- h3_cell como PK se resuelve con un índice simple.
--
-- active=false es un soft delete: fare_quotes.zone_fare_rule_id (siguiente
-- migración) puede referenciar una fila ya desactivada, así que no se borra
-- físicamente y no se pierde la trazabilidad histórica.
CREATE TABLE trips.zone_fare_rules (
    zone_id                              UUID PRIMARY KEY,
    name                                 VARCHAR NOT NULL,
    included_km                          DOUBLE PRECISION NOT NULL,
    day_min_fare_usd                     DOUBLE PRECISION NOT NULL,
    day_extra_rate_per_km_usd            DOUBLE PRECISION NOT NULL,
    night_min_fare_usd                   DOUBLE PRECISION NOT NULL,
    night_extra_rate_per_km_usd          DOUBLE PRECISION NOT NULL,
    early_morning_min_fare_usd           DOUBLE PRECISION NOT NULL,
    early_morning_extra_rate_per_km_usd  DOUBLE PRECISION NOT NULL,
    active                               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trips.zone_fare_cells (
    h3_cell VARCHAR PRIMARY KEY,
    zone_id UUID NOT NULL REFERENCES trips.zone_fare_rules (zone_id)
);

CREATE INDEX idx_zone_fare_cells_zone_id ON trips.zone_fare_cells (zone_id);

-- +goose Down
DROP INDEX IF EXISTS trips.idx_zone_fare_cells_zone_id;
DROP TABLE IF EXISTS trips.zone_fare_cells;
DROP TABLE IF EXISTS trips.zone_fare_rules;
