-- +goose Up
DROP INDEX IF EXISTS orders.idx_orders_pending_unassigned_alert;

CREATE INDEX IF NOT EXISTS idx_orders_pending_unassigned_alert
    ON orders.orders (pending_at)
    WHERE status = 'Pending'
      AND pending_at IS NOT NULL
      AND pending_alert_minute < 8;

-- +goose Down
DROP INDEX IF EXISTS orders.idx_orders_pending_unassigned_alert;

CREATE INDEX IF NOT EXISTS idx_orders_pending_unassigned_alert
    ON orders.orders (pending_at)
    WHERE status = 'Pending'
      AND pending_at IS NOT NULL
      AND pending_alert_minute < 4;
