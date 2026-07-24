-- +goose Up
-- Evoluciona el débito de devolución: hold (pending) → processing (accepted) → payout (completed).
-- Datos legacy: payout solo es válido si la solicitud ya está completed.

UPDATE finance.transactions t
SET movement_type = 'balance_refund_hold'
FROM finance.balance_refund_requests r
WHERE t.refund_request_id = r.id
  AND t.amount < 0
  AND t.movement_type = 'balance_refund_payout'
  AND r.status IN ('pending', 'rejected');

UPDATE finance.transactions t
SET movement_type = 'balance_refund_processing'
FROM finance.balance_refund_requests r
WHERE t.refund_request_id = r.id
  AND t.amount < 0
  AND t.movement_type = 'balance_refund_payout'
  AND r.status = 'accepted';

DROP INDEX IF EXISTS finance.idx_finance_tx_refund_payout_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_tx_refund_reservation_unique
    ON finance.transactions (refund_request_id)
    WHERE refund_request_id IS NOT NULL
      AND movement_type IN (
          'balance_refund_hold',
          'balance_refund_processing',
          'balance_refund_payout'
      );

-- +goose Down
DROP INDEX IF EXISTS finance.idx_finance_tx_refund_reservation_unique;

-- Revertir tipos intermedios a payout (comportamiento previo).
UPDATE finance.transactions
SET movement_type = 'balance_refund_payout'
WHERE amount < 0
  AND movement_type IN ('balance_refund_hold', 'balance_refund_processing');

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_tx_refund_payout_unique
    ON finance.transactions (refund_request_id)
    WHERE refund_request_id IS NOT NULL
      AND movement_type = 'balance_refund_payout';
