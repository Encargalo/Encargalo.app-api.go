-- +goose Up
ALTER TABLE orders.orders
    ADD COLUMN IF NOT EXISTS pending_alert_minute INT NOT NULL DEFAULT 0;

-- Migrate one-shot alerts already sent → treat as minute 1 so we don't re-spam.
UPDATE orders.orders
SET pending_alert_minute = 1
WHERE pending_alerted_at IS NOT NULL
  AND pending_alert_minute = 0;

DROP INDEX IF EXISTS orders.idx_orders_pending_unassigned_alert;
ALTER TABLE orders.orders
    DROP COLUMN IF EXISTS pending_alerted_at;

CREATE INDEX IF NOT EXISTS idx_orders_pending_unassigned_alert
    ON orders.orders (pending_at)
    WHERE status = 'Pending'
      AND pending_at IS NOT NULL
      AND pending_alert_minute < 4;

-- +goose Down
DROP INDEX IF EXISTS orders.idx_orders_pending_unassigned_alert;

ALTER TABLE orders.orders
    ADD COLUMN IF NOT EXISTS pending_alerted_at TIMESTAMPTZ;

UPDATE orders.orders
SET pending_alerted_at = now()
WHERE pending_alert_minute > 0
  AND pending_alerted_at IS NULL;

ALTER TABLE orders.orders
    DROP COLUMN IF EXISTS pending_alert_minute;

CREATE INDEX IF NOT EXISTS idx_orders_pending_unassigned_alert
    ON orders.orders (pending_at)
    WHERE status = 'Pending'
      AND pending_at IS NOT NULL
      AND pending_alerted_at IS NULL;
