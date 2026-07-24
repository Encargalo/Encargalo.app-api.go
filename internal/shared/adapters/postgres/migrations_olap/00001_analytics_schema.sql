-- +goose Up
CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.etl_sync_logs
(
    entity         TEXT PRIMARY KEY,
    last_synced_to TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.dim_shop
(
    shop_id        UUID PRIMARY KEY,
    name           TEXT        NOT NULL,
    tag            TEXT,
    address        TEXT,
    type           TEXT,
    license_status TEXT,
    opened         BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.dim_user
(
    user_id       UUID PRIMARY KEY,
    name          TEXT        NOT NULL,
    sur_name      TEXT,
    phone         TEXT,
    email         TEXT,
    roles         TEXT[]      NOT NULL DEFAULT '{}',
    enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    registered_at TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    synced_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dim_user_registered_at ON analytics.dim_user (registered_at);
CREATE INDEX IF NOT EXISTS idx_dim_user_roles ON analytics.dim_user USING GIN (roles);

CREATE TABLE IF NOT EXISTS analytics.dim_rider
(
    rider_id      UUID PRIMARY KEY,
    name          TEXT        NOT NULL,
    sur_name      TEXT,
    phone         TEXT,
    email         TEXT,
    status        TEXT        NOT NULL,
    vehicle_type  TEXT,
    registered_at TIMESTAMPTZ NOT NULL,
    activated_at  TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ NOT NULL,
    synced_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dim_rider_registered_at ON analytics.dim_rider (registered_at);
CREATE INDEX IF NOT EXISTS idx_dim_rider_status ON analytics.dim_rider (status);

CREATE TABLE IF NOT EXISTS analytics.cube_order
(
    order_id             UUID PRIMARY KEY,
    shop_id              UUID           NOT NULL,
    customer_id          UUID           NOT NULL,
    rider_id             UUID,
    status               TEXT           NOT NULL,
    method_payment       TEXT,
    total_price          NUMERIC(12, 2) NOT NULL DEFAULT 0,
    delivery_fee         NUMERIC(12, 2) NOT NULL DEFAULT 0,
    gmv                  NUMERIC(12, 2) NOT NULL DEFAULT 0,
    promo_subsidy_amount NUMERIC(12, 2) NOT NULL DEFAULT 0,
    is_launch_promo      BOOLEAN        NOT NULL DEFAULT FALSE,
    is_zero_commission   BOOLEAN        NOT NULL DEFAULT FALSE,
    order_date           DATE           NOT NULL,
    created_at           TIMESTAMPTZ    NOT NULL,
    synced_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cube_order_created_at ON analytics.cube_order (created_at);
CREATE INDEX IF NOT EXISTS idx_cube_order_order_date ON analytics.cube_order (order_date);
CREATE INDEX IF NOT EXISTS idx_cube_order_status ON analytics.cube_order (status);
CREATE INDEX IF NOT EXISTS idx_cube_order_shop_id ON analytics.cube_order (shop_id);
CREATE INDEX IF NOT EXISTS idx_cube_order_customer_id ON analytics.cube_order (customer_id);
CREATE INDEX IF NOT EXISTS idx_cube_order_rider_id ON analytics.cube_order (rider_id)
    WHERE rider_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS analytics.cube_order;
DROP TABLE IF EXISTS analytics.dim_rider;
DROP TABLE IF EXISTS analytics.dim_user;
DROP TABLE IF EXISTS analytics.dim_shop;
DROP TABLE IF EXISTS analytics.etl_sync_logs;
DROP SCHEMA IF EXISTS analytics;
