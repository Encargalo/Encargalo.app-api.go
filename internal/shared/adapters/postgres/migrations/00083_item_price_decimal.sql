-- +goose Up
-- Precios en USD con centavos: los productos de víveres valen menos de $1.
ALTER TABLE products.items
    ALTER COLUMN price TYPE NUMERIC(12,2);

ALTER TABLE orders.order_items
    ALTER COLUMN unit_price TYPE NUMERIC(12,2),
    ALTER COLUMN total_price TYPE NUMERIC(12,2);

ALTER TABLE orders.additions_order
    ALTER COLUMN unit_price TYPE NUMERIC(12,2),
    ALTER COLUMN total_price TYPE NUMERIC(12,2);

-- +goose Down
ALTER TABLE orders.additions_order
    ALTER COLUMN unit_price TYPE INTEGER USING round(unit_price)::integer,
    ALTER COLUMN total_price TYPE INTEGER USING round(total_price)::integer;

ALTER TABLE orders.order_items
    ALTER COLUMN unit_price TYPE INTEGER USING round(unit_price)::integer,
    ALTER COLUMN total_price TYPE INTEGER USING round(total_price)::integer;

ALTER TABLE products.items
    ALTER COLUMN price TYPE INTEGER USING round(price)::integer;
