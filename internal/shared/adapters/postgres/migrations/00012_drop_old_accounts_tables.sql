-- +goose Up
-- +goose StatementBegin

-- Eliminar tablas de accounts de los schemas antiguos
-- (los datos ya fueron migrados a users.accounts en 00010)
DROP TABLE IF EXISTS customers.accounts;
DROP TABLE IF EXISTS employees.accounts;
DROP TABLE IF EXISTS owners.accounts;

-- Eliminar los schemas que quedaron vacíos
DROP SCHEMA IF EXISTS customers;
DROP SCHEMA IF EXISTS employees;
DROP SCHEMA IF EXISTS owners;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Recrear schemas
CREATE SCHEMA IF NOT EXISTS customers;
CREATE SCHEMA IF NOT EXISTS employees;
CREATE SCHEMA IF NOT EXISTS owners;

-- Recrear tabla owners.accounts
CREATE TABLE owners.accounts
(
    id         UUID      DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    name       VARCHAR   NOT NULL,
    email      VARCHAR   NOT NULL UNIQUE,
    phone      VARCHAR   NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Recrear tabla customers.accounts
CREATE TABLE customers.accounts
(
    id                UUID      DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    name              VARCHAR   NOT NULL,
    sur_name          VARCHAR   NOT NULL,
    phone             VARCHAR   NOT NULL,
    email             VARCHAR,
    password          VARCHAR   NOT NULL,
    birthday_date     DATE      NOT NULL,
    activation_status VARCHAR   DEFAULT 'in progress',
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW(),
    deleted_at        TIMESTAMP
);

-- Recrear tabla employees.accounts
CREATE TABLE employees.accounts
(
    id         UUID      DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id    UUID      NOT NULL,
    name       VARCHAR   NOT NULL,
    email      VARCHAR   NOT NULL,
    phone      VARCHAR   NOT NULL,
    password   VARCHAR   NOT NULL,
    role       VARCHAR   NOT NULL,
    enabled    BOOLEAN   DEFAULT TRUE,
    created_by UUID      NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- +goose StatementEnd
