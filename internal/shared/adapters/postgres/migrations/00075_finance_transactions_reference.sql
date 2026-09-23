-- +goose Up
-- reference_id/reference_type generalizan order_id para customer (users) y
-- staff_payout (finance). order_id NO se elimina: lo siguen usando las filas
-- entity_type='rider' (internal/riders).
ALTER TABLE finance.transactions
    ADD COLUMN IF NOT EXISTS reference_id   UUID,
    ADD COLUMN IF NOT EXISTS reference_type TEXT;

CREATE INDEX IF NOT EXISTS idx_finance_transactions_reference
    ON finance.transactions (reference_type, reference_id);

-- +goose Down
DROP INDEX IF EXISTS finance.idx_finance_transactions_reference;
ALTER TABLE finance.transactions
    DROP COLUMN IF EXISTS reference_type,
    DROP COLUMN IF EXISTS reference_id;
