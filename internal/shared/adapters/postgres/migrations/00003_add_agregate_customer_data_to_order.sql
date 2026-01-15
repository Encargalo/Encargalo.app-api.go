-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

alter table orders.orders add column customer_name varchar(255), add column phone_customer varchar(13);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
