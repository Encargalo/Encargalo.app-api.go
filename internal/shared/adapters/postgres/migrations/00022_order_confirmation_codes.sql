-- +goose Up
ALTER TABLE orders.orders
  ADD COLUMN pickup_code            VARCHAR(6),
  ADD COLUMN delivery_code          VARCHAR(6),
  ADD COLUMN pickup_code_used_at    TIMESTAMPTZ,
  ADD COLUMN delivery_code_used_at  TIMESTAMPTZ;

-- +goose Down
ALTER TABLE orders.orders
  DROP COLUMN IF EXISTS pickup_code,
  DROP COLUMN IF EXISTS delivery_code,
  DROP COLUMN IF EXISTS pickup_code_used_at,
  DROP COLUMN IF EXISTS delivery_code_used_at;
