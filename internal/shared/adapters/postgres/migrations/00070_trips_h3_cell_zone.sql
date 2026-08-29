-- +goose Up
-- pickup_h3_cell_zone/dropoff_h3_cell_zone son distintos de pickup_h3_cell:
-- pickup_h3_cell usa geo.DriverMatchResolution (8, matching en tiempo real);
-- estos usan geo.ZoneAnalyticsResolution (7, agregación de zonas para
-- heatmaps), mismo patrón que orders.orders.h3_cell.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS pickup_h3_cell_zone  VARCHAR,
    ADD COLUMN IF NOT EXISTS dropoff_h3_cell_zone VARCHAR;

CREATE INDEX IF NOT EXISTS idx_trips_pickup_h3_cell_zone ON trips.trips (pickup_h3_cell_zone)
    WHERE pickup_h3_cell_zone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trips_dropoff_h3_cell_zone ON trips.trips (dropoff_h3_cell_zone)
    WHERE dropoff_h3_cell_zone IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS trips.idx_trips_dropoff_h3_cell_zone;
DROP INDEX IF EXISTS trips.idx_trips_pickup_h3_cell_zone;
ALTER TABLE trips.trips
    DROP COLUMN IF EXISTS dropoff_h3_cell_zone,
    DROP COLUMN IF EXISTS pickup_h3_cell_zone;
