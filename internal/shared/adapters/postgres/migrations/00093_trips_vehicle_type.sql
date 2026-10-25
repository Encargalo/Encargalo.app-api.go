-- +goose Up
-- El vehículo se congela en la cotización y el viaje lo hereda: sin esto un
-- cliente podría cotizar en moto y pedir un carro con la tarifa ya congelada.
ALTER TABLE trips.fare_quotes ADD COLUMN vehicle_type VARCHAR NOT NULL DEFAULT 'moto';
ALTER TABLE trips.trips ADD COLUMN vehicle_type VARCHAR NOT NULL DEFAULT 'moto';

-- +goose Down
ALTER TABLE trips.trips DROP COLUMN vehicle_type;
ALTER TABLE trips.fare_quotes DROP COLUMN vehicle_type;
