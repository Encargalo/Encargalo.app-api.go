-- +goose Up
-- Ninguna query del código actual usa estos 5 índices: idx_trips_status
-- siempre se filtra junto con id = ? (ya resuelto por la PK, ver
-- internal/trips/repo/trips.go); idx_trips_rider_id nunca se filtra solo;
-- los 3 de H3 solo se leen como columna (el matching de riders es en
-- memoria, no SQL, y el heatmap de zonas vive en OLAP vía el ETL, ver
-- internal/analytics/repo/oltp_source.go). Cada uno es puro costo de
-- escritura en cada INSERT y, en el caso de idx_trips_status, en cada
-- transición de estado del viaje (hasta 6 veces por ciclo de vida completo).
DROP INDEX IF EXISTS trips.idx_trips_status;
DROP INDEX IF EXISTS trips.idx_trips_rider_id;
DROP INDEX IF EXISTS trips.idx_trips_pickup_h3_cell;
DROP INDEX IF EXISTS trips.idx_trips_pickup_h3_cell_zone;
DROP INDEX IF EXISTS trips.idx_trips_dropoff_h3_cell_zone;

-- +goose Down
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips.trips (status);
CREATE INDEX IF NOT EXISTS idx_trips_rider_id ON trips.trips (rider_id)
    WHERE rider_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trips_pickup_h3_cell ON trips.trips (pickup_h3_cell)
    WHERE pickup_h3_cell IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trips_pickup_h3_cell_zone ON trips.trips (pickup_h3_cell_zone)
    WHERE pickup_h3_cell_zone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trips_dropoff_h3_cell_zone ON trips.trips (dropoff_h3_cell_zone)
    WHERE dropoff_h3_cell_zone IS NOT NULL;
