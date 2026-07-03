-- +goose Up
UPDATE orders.orders
  SET pickup_code = NULL
  WHERE LENGTH(pickup_code) > 4;

UPDATE orders.orders
  SET delivery_code = NULL
  WHERE LENGTH(delivery_code) > 4;

ALTER TABLE orders.orders
  ALTER COLUMN pickup_code TYPE VARCHAR(4),
  ALTER COLUMN delivery_code TYPE VARCHAR(4);

-- +goose Down
ALTER TABLE orders.orders
  ALTER COLUMN pickup_code TYPE VARCHAR(6),
  ALTER COLUMN delivery_code TYPE VARCHAR(6);
