-- +goose Up
ALTER TABLE orders.payment_reviews
    ADD COLUMN IF NOT EXISTS last_alerted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_payment_reviews_alert
    ON orders.payment_reviews (status, last_alerted_at)
    WHERE status = 'pending_review';

-- +goose Down
DROP INDEX IF EXISTS orders.idx_payment_reviews_alert;
ALTER TABLE orders.payment_reviews
    DROP COLUMN IF EXISTS last_alerted_at;
