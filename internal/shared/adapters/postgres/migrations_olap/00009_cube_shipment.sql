-- +goose Up
-- pickup_h3_cell/dropoff_h3_cell son resolución de zona (ZoneAnalyticsResolution), copiados de shipments.shipments.pickup_h3_cell_zone/dropoff_h3_cell_zone.
CREATE TABLE IF NOT EXISTS analytics.cube_shipment
(
    shipment_id     UUID           PRIMARY KEY,
    quote_id        UUID,
    customer_id     UUID           NOT NULL,
    rider_id        UUID,
    status          TEXT           NOT NULL,
    fare            NUMERIC(12, 2),
    method_payment  TEXT,
    pickup_h3_cell  TEXT,
    dropoff_h3_cell TEXT,
    shipment_date   DATE           NOT NULL,
    created_at      TIMESTAMPTZ    NOT NULL,
    synced_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cube_shipment_created_at ON analytics.cube_shipment (created_at);
CREATE INDEX IF NOT EXISTS idx_cube_shipment_shipment_date ON analytics.cube_shipment (shipment_date);
CREATE INDEX IF NOT EXISTS idx_cube_shipment_status ON analytics.cube_shipment (status);
CREATE INDEX IF NOT EXISTS idx_cube_shipment_customer_id ON analytics.cube_shipment (customer_id);
CREATE INDEX IF NOT EXISTS idx_cube_shipment_rider_id ON analytics.cube_shipment (rider_id)
    WHERE rider_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS analytics.cube_shipment;
