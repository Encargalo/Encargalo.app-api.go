-- +goose Up
ALTER TABLE orders.orders
  ADD COLUMN rider_id           UUID,
  ADD COLUMN rider_accepted_at  TIMESTAMPTZ;

-- +goose Down
ALTER TABLE orders.orders
  DROP COLUMN IF EXISTS rider_id,
  DROP COLUMN IF EXISTS rider_accepted_at;
