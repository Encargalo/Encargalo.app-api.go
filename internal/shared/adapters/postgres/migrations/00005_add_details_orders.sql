-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd
alter table orders.orders add column details jsonb; 
-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
