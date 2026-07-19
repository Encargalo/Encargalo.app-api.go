-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_tx_refund_payout_unique
    ON finance.transactions (refund_request_id)
    WHERE refund_request_id IS NOT NULL
      AND movement_type = 'balance_refund_payout';

-- +goose Down
DROP INDEX IF EXISTS finance.idx_finance_tx_refund_payout_unique;
