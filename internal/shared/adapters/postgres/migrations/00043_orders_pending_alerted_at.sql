-- +goose Up
ALTER TABLE orders.orders
    ADD COLUMN IF NOT EXISTS pending_alerted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_orders_pending_unassigned_alert
    ON orders.orders (pending_at)
    WHERE status = 'Pending'
      AND pending_at IS NOT NULL
      AND pending_alerted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS orders.idx_orders_pending_unassigned_alert;
ALTER TABLE orders.orders
    DROP COLUMN IF EXISTS pending_alerted_at;
