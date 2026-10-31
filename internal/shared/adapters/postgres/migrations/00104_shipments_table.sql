-- +goose Up
CREATE SCHEMA IF NOT EXISTS shipments;

CREATE TABLE IF NOT EXISTS shipments.shipments
(
    id                    UUID             NOT NULL PRIMARY KEY,
    customer_id           UUID             NOT NULL REFERENCES users.accounts (id),
    quote_id              UUID,
    rider_id              UUID,
    status                VARCHAR          NOT NULL DEFAULT 'requested',
    pickup_address        VARCHAR          NOT NULL,
    pickup_latitude       DOUBLE PRECISION NOT NULL,
    pickup_longitude      DOUBLE PRECISION NOT NULL,
    pickup_h3_cell_zone   VARCHAR,
    dropoff_address       VARCHAR          NOT NULL,
    dropoff_latitude      DOUBLE PRECISION NOT NULL,
    dropoff_longitude     DOUBLE PRECISION NOT NULL,
    dropoff_h3_cell_zone  VARCHAR,
    sender_is_customer    BOOLEAN          NOT NULL DEFAULT TRUE,
    sender_name           VARCHAR          NOT NULL,
    sender_phone          VARCHAR          NOT NULL,
    recipient_is_customer BOOLEAN          NOT NULL DEFAULT TRUE,
    recipient_name        VARCHAR          NOT NULL,
    recipient_phone       VARCHAR          NOT NULL,
    package_description   VARCHAR          NOT NULL,
    fare                  NUMERIC(12, 2),
    method_payment        VARCHAR          NOT NULL DEFAULT '',
    bcv_rate              NUMERIC(14, 4),
    balance_paid_bs       NUMERIC(14, 2),
    payment               JSONB,
    cancel_reason         VARCHAR,
    -- Siempre NOT NULL: a diferencia de trips.trips, los códigos se generan en
    -- la creación del envío, no al aceptar/iniciar, y siempre se exponen al
    -- cliente dueño del envío sin importar el estado.
    pickup_code           VARCHAR          NOT NULL,
    dropoff_code          VARCHAR          NOT NULL,
    pickup_code_used_at   TIMESTAMPTZ,
    dropoff_code_used_at  TIMESTAMPTZ,
    requested_at          TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    accepted_at           TIMESTAMPTZ,
    delivered_at          TIMESTAMPTZ,
    cancelled_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shipments_customer_id ON shipments.shipments (customer_id);
CREATE INDEX IF NOT EXISTS idx_shipments_status ON shipments.shipments (status);
CREATE INDEX IF NOT EXISTS idx_shipments_rider_id ON shipments.shipments (rider_id) WHERE rider_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS shipments.fare_quotes
(
    quote_id             UUID             NOT NULL PRIMARY KEY,
    customer_id          UUID             NOT NULL REFERENCES users.accounts (id),
    pickup_latitude      DOUBLE PRECISION NOT NULL,
    pickup_longitude     DOUBLE PRECISION NOT NULL,
    dropoff_latitude     DOUBLE PRECISION NOT NULL,
    dropoff_longitude    DOUBLE PRECISION NOT NULL,
    pickup_h3_cell_zone  VARCHAR,
    dropoff_h3_cell_zone VARCHAR,
    distance_meters      DOUBLE PRECISION NOT NULL,
    duration_min         BIGINT           NOT NULL,
    fare_usd             NUMERIC(12, 2)   NOT NULL,
    fare_bs              NUMERIC(14, 2)   NOT NULL,
    bcv_rate             NUMERIC(14, 4)   NOT NULL,
    fare_band            VARCHAR          NOT NULL,
    created_at           TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shipment_fare_quotes_customer_id ON shipments.fare_quotes (customer_id);

-- +goose Down
DROP TABLE IF EXISTS shipments.fare_quotes;
DROP TABLE IF EXISTS shipments.shipments;
DROP SCHEMA IF EXISTS shipments;
