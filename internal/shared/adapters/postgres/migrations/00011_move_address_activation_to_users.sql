-- +goose Up
-- +goose StatementBegin

ALTER TABLE customers.address SET SCHEMA users;
ALTER TABLE customers.activate_account SET SCHEMA users;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE users.address SET SCHEMA customers;
ALTER TABLE users.activate_account SET SCHEMA customers;

-- +goose StatementEnd
