-- +goose Up
CREATE SCHEMA IF NOT EXISTS finance;

CREATE TABLE finance.balances (
    entity_id   UUID          NOT NULL,
    entity_type TEXT          NOT NULL,
    amount      NUMERIC(12,2) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, entity_type)
);

CREATE TABLE finance.transactions (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id      UUID          NOT NULL,
    entity_type    TEXT          NOT NULL,
    order_id       UUID,
    amount         NUMERIC(12,2) NOT NULL,
    movement_type  TEXT          NOT NULL,
    payment_method TEXT,
    distance_km    NUMERIC(8,3),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_finance_transactions_entity
    ON finance.transactions (entity_id, entity_type, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS finance.transactions;
DROP TABLE IF EXISTS finance.balances;
DROP SCHEMA IF EXISTS finance;
