-- +goose Up
ALTER TABLE orders.orders
    DROP CONSTRAINT IF EXISTS orders_method_payment_check;

UPDATE orders.orders SET method_payment = 'EfectivoBs'    WHERE method_payment = 'Efectivo';
UPDATE orders.orders SET method_payment = 'PagoMovil'     WHERE method_payment = 'Nequi';

ALTER TABLE orders.orders
    ADD CONSTRAINT orders_method_payment_check
    CHECK (method_payment = ANY (ARRAY['EfectivoBs', 'EfectivoDolar', 'PagoMovil']));

-- +goose Down
ALTER TABLE orders.orders
    DROP CONSTRAINT IF EXISTS orders_method_payment_check;

UPDATE orders.orders SET method_payment = 'Efectivo' WHERE method_payment IN ('EfectivoBs', 'EfectivoDolar');
UPDATE orders.orders SET method_payment = 'Nequi'    WHERE method_payment = 'PagoMovil';

ALTER TABLE orders.orders
    ADD CONSTRAINT orders_method_payment_check
    CHECK (method_payment = ANY (ARRAY['Efectivo', 'Nequi']));
