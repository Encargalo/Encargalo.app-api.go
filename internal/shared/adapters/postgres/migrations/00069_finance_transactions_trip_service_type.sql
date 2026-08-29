-- +goose Up
-- trip_id enlaza el movimiento a trips.trips para ride_bank de viajes (RecordTripEarning).
-- service_type distingue viaje/domicilio en movimientos ride_bank/trip_bank; nulo en
-- movimientos que no son de ganancia por servicio (withdrawal, adjustment_*, staff_payout).
ALTER TABLE finance.transactions
    ADD COLUMN IF NOT EXISTS trip_id      UUID REFERENCES trips.trips (id),
    ADD COLUMN IF NOT EXISTS service_type TEXT;

-- +goose Down
ALTER TABLE finance.transactions
    DROP COLUMN IF EXISTS service_type,
    DROP COLUMN IF EXISTS trip_id;
