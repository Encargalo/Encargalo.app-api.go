-- +goose Up
CREATE SCHEMA IF NOT EXISTS payments;

CREATE TABLE payments.reviews (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id        UUID          NOT NULL,
    reference_type      TEXT          NOT NULL DEFAULT 'order',
    proof_url           TEXT          NOT NULL,
    expected_amount_bs  NUMERIC(14,2) NOT NULL,
    observed_amount_bs  NUMERIC(14,2),
    difference_bs       NUMERIC(14,2),
    bcv_rate            NUMERIC(14,6) NOT NULL DEFAULT 0,
    status              TEXT          NOT NULL DEFAULT 'pending_review',
    rejection_reason    TEXT,
    staff_id            UUID REFERENCES users.accounts (id),
    credited_bs         NUMERIC(12,2),
    reference_last4     CHAR(4),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    last_alerted_at     TIMESTAMPTZ
);

CREATE INDEX idx_payments_reviews_status ON payments.reviews (status);
CREATE INDEX idx_payments_reviews_reference ON payments.reviews (reference_type, reference_id, created_at DESC);
CREATE INDEX idx_payments_reviews_alert ON payments.reviews (status, last_alerted_at) WHERE status = 'pending_review';
CREATE UNIQUE INDEX idx_payments_reviews_reference_last4_unique
    ON payments.reviews (reference_last4)
    WHERE reference_last4 IS NOT NULL AND status IN ('approved', 'overpaid', 'rejected_underpaid');

-- Datos existentes no importan (sin operación real todavía).
DROP TABLE IF EXISTS orders.payment_reviews;

-- +goose Down
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
    credited_bs         NUMERIC(12,2),
    reference_last4     CHAR(4),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    last_alerted_at     TIMESTAMPTZ
);
CREATE INDEX idx_payment_reviews_status ON orders.payment_reviews (status);
CREATE INDEX idx_payment_reviews_order ON orders.payment_reviews (order_id, created_at DESC);
CREATE UNIQUE INDEX idx_payment_reviews_reference_last4_unique
    ON orders.payment_reviews (reference_last4)
    WHERE reference_last4 IS NOT NULL AND status IN ('approved', 'overpaid', 'rejected_underpaid');
CREATE INDEX idx_payment_reviews_alert ON orders.payment_reviews (status, last_alerted_at) WHERE status = 'pending_review';

DROP SCHEMA IF EXISTS payments CASCADE;
