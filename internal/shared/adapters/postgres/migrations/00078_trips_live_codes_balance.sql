-- +goose Up
-- Códigos de 4 dígitos para confirmar subida y bajada, y el monto de balance
-- debitado cuando el viaje se paga entero con saldo (sin comprobante).
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS pickup_code VARCHAR(4),
    ADD COLUMN IF NOT EXISTS dropoff_code VARCHAR(4),
    ADD COLUMN IF NOT EXISTS pickup_code_used_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS dropoff_code_used_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS balance_paid_bs NUMERIC(14, 2);

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS pickup_code,
    DROP COLUMN IF EXISTS dropoff_code,
    DROP COLUMN IF EXISTS pickup_code_used_at,
    DROP COLUMN IF EXISTS dropoff_code_used_at,
    DROP COLUMN IF EXISTS balance_paid_bs;
