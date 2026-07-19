-- +goose Up
ALTER TABLE users.accounts
    ADD COLUMN IF NOT EXISTS pago_movil JSONB;

ALTER TABLE finance.transactions
    ADD COLUMN IF NOT EXISTS refundable_pm_usd NUMERIC(12,2),
    ADD COLUMN IF NOT EXISTS refund_request_id UUID;

CREATE TABLE IF NOT EXISTS finance.balance_refund_requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id      UUID          NOT NULL REFERENCES users.accounts (id),
    transaction_id   UUID          NOT NULL REFERENCES finance.transactions (id),
    order_id         UUID,
    amount_usd       NUMERIC(12,2) NOT NULL,
    status           TEXT          NOT NULL DEFAULT 'pending',
    pm_phone         TEXT          NOT NULL,
    pm_bank_code     TEXT          NOT NULL,
    pm_national_id   TEXT          NOT NULL,
    proof_url        TEXT,
    staff_id         UUID REFERENCES users.accounts (id),
    rejection_reason TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ   NOT NULL,
    accepted_at      TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    resolved_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_balance_refund_requests_tx_active
    ON finance.balance_refund_requests (transaction_id)
    WHERE status IN ('pending', 'accepted');

CREATE INDEX IF NOT EXISTS idx_balance_refund_requests_status
    ON finance.balance_refund_requests (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_balance_refund_requests_customer
    ON finance.balance_refund_requests (customer_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS finance.idx_balance_refund_requests_customer;
DROP INDEX IF EXISTS finance.idx_balance_refund_requests_status;
DROP INDEX IF EXISTS finance.idx_balance_refund_requests_tx_active;
DROP TABLE IF EXISTS finance.balance_refund_requests;
ALTER TABLE finance.transactions
    DROP COLUMN IF EXISTS refund_request_id,
    DROP COLUMN IF EXISTS refundable_pm_usd;
ALTER TABLE users.accounts
    DROP COLUMN IF EXISTS pago_movil;
