-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

-- SE AGREGA LAS FUNCIONES PARA UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- SE AGREGAN LOS SCHEMAS EXISTENTES
CREATE SCHEMA IF NOT EXISTS business;
CREATE SCHEMA IF NOT EXISTS customers;
CREATE SCHEMA IF NOT EXISTS employees;
CREATE SCHEMA IF NOT EXISTS licenses;
CREATE SCHEMA IF NOT EXISTS orders;
CREATE SCHEMA IF NOT EXISTS owners;
CREATE SCHEMA IF NOT EXISTS products;
CREATE SCHEMA IF NOT EXISTS sessions;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
