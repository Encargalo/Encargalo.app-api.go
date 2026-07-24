-- +goose Up
ALTER TABLE orders.orders
    ADD COLUMN IF NOT EXISTS pending_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_orders_pending_at_stale
    ON orders.orders (pending_at)
    WHERE status = 'Pending' AND pending_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS orders.idx_orders_pending_at_stale;
ALTER TABLE orders.orders
    DROP COLUMN IF EXISTS pending_at;
