-- +goose Up
-- Los balances/movimientos de customers y riders pasan a Bs (VES); shops y
-- platform_subsidy se mantienen en USD. currency documenta la unidad de
-- `amount` por fila; bcv_rate congela la tasa usada para convertir el monto
-- original en USD (precio del pedido) al Bs acreditado/debitado, para poder
-- reconstruir el USD equivalente de un movimiento pasado (ver
-- customerFinanceApp.amountToUSD).
ALTER TABLE finance.balances
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'USD';

ALTER TABLE finance.transactions
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS bcv_rate NUMERIC(14,6);

UPDATE finance.balances
    SET currency = 'VES'
    WHERE entity_type IN ('customer', 'rider');

UPDATE finance.transactions
    SET currency = 'VES'
    WHERE entity_type IN ('customer', 'rider');

ALTER TABLE finance.transactions
    RENAME COLUMN refundable_pm_usd TO refundable_pm_bs;

ALTER TABLE finance.balance_refund_requests
    RENAME COLUMN amount_usd TO amount_bs;

ALTER TABLE orders.payment_reviews
    RENAME COLUMN credited_usd TO credited_bs;

-- +goose Down
ALTER TABLE orders.payment_reviews
    RENAME COLUMN credited_bs TO credited_usd;

ALTER TABLE finance.balance_refund_requests
    RENAME COLUMN amount_bs TO amount_usd;

ALTER TABLE finance.transactions
    RENAME COLUMN refundable_pm_bs TO refundable_pm_usd;

ALTER TABLE finance.transactions
    DROP COLUMN IF EXISTS bcv_rate,
    DROP COLUMN IF EXISTS currency;

ALTER TABLE finance.balances
    DROP COLUMN IF EXISTS currency;
