-- +goose Up
-- balance_refund_requests.order_id solo lo usa internal/users.
ALTER TABLE finance.balance_refund_requests
    ADD COLUMN IF NOT EXISTS reference_type TEXT NOT NULL DEFAULT 'order';

ALTER TABLE finance.balance_refund_requests
    RENAME COLUMN order_id TO reference_id;

-- +goose Down
ALTER TABLE finance.balance_refund_requests
    RENAME COLUMN reference_id TO order_id;
ALTER TABLE finance.balance_refund_requests
    DROP COLUMN IF EXISTS reference_type;
