-- +goose Up
ALTER TABLE finance.balance_refund_requests
    ADD COLUMN IF NOT EXISTS last_alerted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_balance_refund_requests_alert
    ON finance.balance_refund_requests (status, last_alerted_at)
    WHERE status IN ('pending', 'accepted');

-- +goose Down
DROP INDEX IF EXISTS finance.idx_balance_refund_requests_alert;
ALTER TABLE finance.balance_refund_requests
    DROP COLUMN IF EXISTS last_alerted_at;
