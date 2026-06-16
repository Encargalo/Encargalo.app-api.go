-- +goose Up
ALTER TABLE business.shops DROP COLUMN IF EXISTS maximum_delivery;
ALTER TABLE business.shops DROP COLUMN IF EXISTS minimun_delivery;
ALTER TABLE business.shops DROP COLUMN IF EXISTS coverage;
ALTER TABLE orders.orders ALTER COLUMN delivery_fee TYPE NUMERIC(10,2);
ALTER TABLE orders.orders ALTER COLUMN total_price TYPE NUMERIC(10,2);

-- +goose Down
ALTER TABLE business.shops ADD COLUMN IF NOT EXISTS maximum_delivery INTEGER NOT NULL DEFAULT 0;
ALTER TABLE business.shops ADD COLUMN IF NOT EXISTS minimun_delivery INTEGER NOT NULL DEFAULT 0;
ALTER TABLE business.shops ADD COLUMN IF NOT EXISTS coverage REAL NOT NULL DEFAULT 0;
ALTER TABLE orders.orders ALTER COLUMN delivery_fee TYPE INTEGER USING delivery_fee::INTEGER;
ALTER TABLE orders.orders ALTER COLUMN total_price TYPE INTEGER USING total_price::INTEGER;
