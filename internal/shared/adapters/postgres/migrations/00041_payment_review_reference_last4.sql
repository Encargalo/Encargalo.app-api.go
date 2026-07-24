-- +goose Up
ALTER TABLE orders.payment_reviews
    ADD COLUMN IF NOT EXISTS reference_last4 CHAR(4);

-- Unicidad solo entre comprobantes "consumidos" (aceptados o underpaid).
-- rejected_invalid no cuenta (foto basura / duplicado auto-rechazado).
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_reviews_reference_last4_unique
    ON orders.payment_reviews (reference_last4)
    WHERE reference_last4 IS NOT NULL
      AND status IN ('approved', 'overpaid', 'rejected_underpaid');

-- +goose Down
DROP INDEX IF EXISTS orders.idx_payment_reviews_reference_last4_unique;
ALTER TABLE orders.payment_reviews
    DROP COLUMN IF EXISTS reference_last4;
