-- +goose Up
-- Parcial, no el idx_trips_status que 00071 tumbó por costoso: este solo
-- indexa las filas en 'searching' (pocas a la vez), así que el barrido de
-- rescate de #359 (ListSearchingTripIDs) no hace table scan y las otras 5
-- transiciones de estado del viaje no pagan el costo de escritura.
CREATE INDEX IF NOT EXISTS idx_trips_status_searching ON trips.trips (status)
    WHERE status = 'searching';

-- +goose Down
DROP INDEX IF EXISTS trips.idx_trips_status_searching;
