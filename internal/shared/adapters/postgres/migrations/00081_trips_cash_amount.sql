-- +goose Up
-- Parte en efectivo de un viaje pagado con dos métodos. NULL en viajes anteriores.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS cash_amount_bs NUMERIC;

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS cash_amount_bs;
