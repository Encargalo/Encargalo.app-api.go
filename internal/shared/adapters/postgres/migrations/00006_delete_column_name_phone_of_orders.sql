-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders.orders
    DROP COLUMN customer_name,
    DROP COLUMN phone_customer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders.orders
    ADD COLUMN customer_name TEXT,
    ADD COLUMN phone_customer TEXT;
-- +goose StatementEnd
