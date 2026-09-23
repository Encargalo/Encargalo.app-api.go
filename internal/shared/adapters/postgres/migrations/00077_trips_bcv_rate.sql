-- +goose Up
-- Congela la tasa con la que se calculó el fare en Bs al crear el viaje.
-- PaymentSubject la usa para ExpectedAmountBs sin volver a consultar el BCV.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS bcv_rate NUMERIC(12, 4);

-- +goose Down
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS bcv_rate;
