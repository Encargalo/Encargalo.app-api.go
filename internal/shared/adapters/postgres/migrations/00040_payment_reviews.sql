-- +goose Up
CREATE TABLE IF NOT EXISTS orders.payment_reviews (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID          NOT NULL REFERENCES orders.orders (id),
    proof_url           TEXT          NOT NULL,
    expected_amount_bs  NUMERIC(14,2) NOT NULL,
    observed_amount_bs  NUMERIC(14,2),
    difference_bs       NUMERIC(14,2),
    bcv_rate            NUMERIC(14,6) NOT NULL DEFAULT 0,
    status              TEXT          NOT NULL DEFAULT 'pending_review',
    rejection_reason    TEXT,
    staff_id            UUID REFERENCES users.accounts (id),
    credited_usd        NUMERIC(12,2),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ
);

CREATE INDEX idx_payment_reviews_status
    ON orders.payment_reviews (status);

CREATE INDEX idx_payment_reviews_order
    ON orders.payment_reviews (order_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS orders.payment_reviews;
