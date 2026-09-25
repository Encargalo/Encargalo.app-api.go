-- +goose Up
-- Crédito a la cartera cuando un Pago Móvil aprobado no se convierte en viaje.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS payment JSONB;

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS payment;
