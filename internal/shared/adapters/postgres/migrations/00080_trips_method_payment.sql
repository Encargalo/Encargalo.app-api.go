-- +goose Up
-- El método distingue Pago Móvil, saldo y efectivo al liquidar el viaje.
-- Vacío en filas anteriores: se liquidan como bancarias.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS method_payment TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS method_payment;
