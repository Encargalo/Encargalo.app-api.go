-- +goose Up
-- delivery_fee queda como lo que paga el cliente; el rider cobra sobre delivery_fee + delivery_subsidy_usd.
ALTER TABLE orders.orders
    ADD COLUMN IF NOT EXISTS delivery_subsidy_usd NUMERIC(12, 2) NOT NULL DEFAULT 0;

-- Un solo pedido con envío gratis vivo o completado por cliente; cancelar o que el negocio rechace lo libera.
CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_first_order_delivery ON orders.orders (customer_id)
    WHERE delivery_subsidy_usd > 0 AND status NOT IN ('Cancelled', 'Rejected');

-- +goose Down
DROP INDEX IF EXISTS orders.uq_orders_first_order_delivery;
ALTER TABLE orders.orders DROP COLUMN IF EXISTS delivery_subsidy_usd;
