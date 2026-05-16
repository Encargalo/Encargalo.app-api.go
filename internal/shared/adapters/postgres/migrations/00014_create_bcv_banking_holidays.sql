-- +goose Up
-- +goose StatementBegin

CREATE SCHEMA IF NOT EXISTS bcv;

CREATE TABLE bcv.banking_holidays (
    date DATE PRIMARY KEY
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS bcv.banking_holidays;
DROP SCHEMA IF EXISTS bcv;

-- +goose StatementEnd
