-- +goose Up
-- Distancia (línea recta, km) del rider al pickup en el momento de aceptar el
-- viaje, congelada para liquidar el bono de acercamiento (tripfare.PickupBonus)
-- sin que un recálculo posterior cambie lo que ya se le prometió al rider.
ALTER TABLE trips.trips ADD COLUMN pickup_distance_km DOUBLE PRECISION;

-- +goose Down
ALTER TABLE trips.trips DROP COLUMN pickup_distance_km;
