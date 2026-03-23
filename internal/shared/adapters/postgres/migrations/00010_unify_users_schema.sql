-- +goose Up
-- +goose StatementBegin

-- ======================================================
-- NUEVAS TABLAS
-- ======================================================

CREATE SCHEMA IF NOT EXISTS users;

CREATE TABLE users.accounts
(
    id                UUID      DEFAULT uuid_generate_v4() PRIMARY KEY,
    name              VARCHAR   NOT NULL,
    sur_name          VARCHAR,
    phone             VARCHAR   NOT NULL UNIQUE,
    email             VARCHAR,
    password          VARCHAR,
    birthday_date     DATE,
    activation_status VARCHAR   DEFAULT 'active',
    enabled           BOOLEAN   DEFAULT TRUE,
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW(),
    deleted_at        TIMESTAMP
);

CREATE TABLE users.roles
(
    id   UUID    DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE
);

INSERT INTO users.roles (name)
VALUES ('Customer'),
       ('Order Manager'),
       ('Owner');

CREATE TABLE users.user_roles
(
    id         UUID      DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id    UUID      NOT NULL REFERENCES users.accounts (id),
    role_id    UUID      NOT NULL REFERENCES users.roles (id),
    shop_id    UUID      REFERENCES business.shops (id),
    created_by UUID      REFERENCES users.accounts (id),
    enabled    BOOLEAN   DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (user_id, role_id, shop_id)
);

-- Índice parcial para garantizar unicidad cuando shop_id es NULL (ej: rol Customer)
-- La restricción UNIQUE estándar no cubre NULLs en PostgreSQL
CREATE UNIQUE INDEX user_roles_unique_no_shop
    ON users.user_roles (user_id, role_id)
    WHERE shop_id IS NULL;

-- ======================================================
-- PASO 1: MIGRAR DATOS A users.accounts
-- Orden: customers primero para conservar sus IDs.
-- Owners y employees usan ON CONFLICT (phone) para manejar
-- el caso de una persona que ya existe como customer.
-- ======================================================

-- Customers → users.accounts
-- customers.accounts no tiene UNIQUE en phone, pueden existir duplicados.
-- El primer registro por teléfono gana; los duplicados se omiten.
INSERT INTO users.accounts (id, name, sur_name, phone, email, password, birthday_date,
                            activation_status, enabled, created_at, updated_at, deleted_at)
SELECT DISTINCT ON (phone)
       id,
       name,
       sur_name,
       phone,
       email,
       password,
       birthday_date::DATE,
       activation_status,
       TRUE,
       created_at,
       updated_at,
       deleted_at
FROM customers.accounts
ORDER BY phone, created_at;

-- Owners → users.accounts
-- Si el teléfono ya existe (mismo usuario como customer), se omite la fila.
INSERT INTO users.accounts (id, name, phone, email, enabled, created_at, updated_at, deleted_at)
SELECT id,
       name,
       phone,
       email,
       TRUE,
       created_at,
       updated_at,
       deleted_at
FROM owners.accounts
ON CONFLICT (phone) DO NOTHING;

-- Employees → users.accounts
-- Si el teléfono ya existe (mismo usuario como customer u owner), se omite la fila.
INSERT INTO users.accounts (id, name, phone, email, password, enabled, created_at, updated_at, deleted_at)
SELECT id,
       name,
       phone,
       email,
       password,
       enabled,
       created_at,
       updated_at,
       deleted_at
FROM employees.accounts
ON CONFLICT (phone) DO NOTHING;

-- ======================================================
-- PASO 2: ASIGNAR ROLES EN users.user_roles
-- Se hace ANTES de actualizar los FKs de shops/orders,
-- porque los JOINs usan los IDs originales de owners y shops.
-- Se hace JOIN por phone para obtener el user_id correcto
-- en caso de que haya habido conflictos de teléfono.
-- ======================================================

-- Roles de customers
INSERT INTO users.user_roles (user_id, role_id)
SELECT ua.id, r.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
         CROSS JOIN users.roles r
WHERE r.name = 'Customer'
ON CONFLICT DO NOTHING;

-- Roles de owners (vinculados a su tienda, usando el owner_id original)
INSERT INTO users.user_roles (user_id, role_id, shop_id)
SELECT ua.id, r.id, s.id
FROM owners.accounts oa
         JOIN users.accounts ua ON ua.phone = oa.phone
         JOIN business.shops s ON s.owner_id = oa.id
         CROSS JOIN users.roles r
WHERE r.name = 'Owner'
ON CONFLICT DO NOTHING;

-- Roles de employees
-- Se resuelve created_by buscando al owner creador en users.accounts por teléfono
INSERT INTO users.user_roles (user_id, role_id, shop_id, created_by, enabled)
SELECT ua.id,
       r.id,
       ea.shop_id,
       ua_creator.id,
       ea.enabled
FROM employees.accounts ea
         JOIN users.accounts ua ON ua.phone = ea.phone
         JOIN users.roles r ON r.name = ea.role
         LEFT JOIN owners.accounts oa_creator ON oa_creator.id = ea.created_by
         LEFT JOIN users.accounts ua_creator ON ua_creator.phone = oa_creator.phone
ON CONFLICT (user_id, role_id, shop_id) DO NOTHING;

-- ======================================================
-- PASO 3: DROPEAR FKs VIEJOS ANTES DE LOS UPDATES
-- Necesario porque los UPDATEs pueden asignar IDs de customers
-- a columnas que antes solo aceptaban IDs de owners/customers,
-- lo que violaría los FKs existentes durante el UPDATE.
-- ======================================================

ALTER TABLE orders.orders
    DROP CONSTRAINT IF EXISTS orders_customer_id_fkey;

ALTER TABLE business.shops
    DROP CONSTRAINT IF EXISTS shops_owner_id_fkey;

-- ======================================================
-- PASO 4: ACTUALIZAR REFERENCIAS A users.accounts
-- Para los casos donde owner/customer fue mergeado por teléfono,
-- el user_id en users.accounts es distinto al original.
-- ======================================================

-- orders.orders.customer_id
UPDATE orders.orders o
SET customer_id = ua.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
WHERE o.customer_id = ca.id;

-- customers.address.customer_id (sin FK explícita, solo datos)
UPDATE customers.address addr
SET customer_id = ua.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
WHERE addr.customer_id = ca.id;

-- customers.activate_account.customer_id (sin FK explícita, solo datos)
UPDATE customers.activate_account act
SET customer_id = ua.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
WHERE act.customer_id = ca.id;

-- business.shops.owner_id
-- Si el owner fue mergeado con un customer, su ID en users.accounts
-- es el del customer. Actualizamos la referencia.
UPDATE business.shops s
SET owner_id = ua.id
FROM owners.accounts oa
         JOIN users.accounts ua ON ua.phone = oa.phone
WHERE s.owner_id = oa.id;

-- ======================================================
-- PASO 5: AGREGAR NUEVOS FKs APUNTANDO A users.accounts
-- ======================================================

ALTER TABLE orders.orders
    ADD CONSTRAINT orders_customer_id_fkey
        FOREIGN KEY (customer_id) REFERENCES users.accounts (id);

ALTER TABLE business.shops
    ADD CONSTRAINT shops_owner_id_fkey
        FOREIGN KEY (owner_id) REFERENCES users.accounts (id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

-- Restaurar FKs originales
ALTER TABLE orders.orders
    DROP CONSTRAINT IF EXISTS orders_customer_id_fkey;
ALTER TABLE orders.orders
    ADD CONSTRAINT orders_customer_id_fkey
        FOREIGN KEY (customer_id) REFERENCES customers.accounts (id);

ALTER TABLE business.shops
    DROP CONSTRAINT IF EXISTS shops_owner_id_fkey;
ALTER TABLE business.shops
    ADD CONSTRAINT shops_owner_id_fkey
        FOREIGN KEY (owner_id) REFERENCES owners.accounts (id);

-- Restaurar datos relacionados a los IDs originales de customers
UPDATE customers.address addr
SET customer_id = ca.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
WHERE addr.customer_id = ua.id;

UPDATE customers.activate_account act
SET customer_id = ca.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
WHERE act.customer_id = ua.id;

UPDATE orders.orders o
SET customer_id = ca.id
FROM customers.accounts ca
         JOIN users.accounts ua ON ua.phone = ca.phone
WHERE o.customer_id = ua.id;

UPDATE business.shops s
SET owner_id = oa.id
FROM owners.accounts oa
         JOIN users.accounts ua ON ua.phone = oa.phone
WHERE s.owner_id = ua.id;

-- Eliminar tablas nuevas
DROP INDEX IF EXISTS users.user_roles_unique_no_shop;
DROP TABLE IF EXISTS users.user_roles;
DROP TABLE IF EXISTS users.roles;
DROP TABLE IF EXISTS users.accounts;
DROP SCHEMA IF EXISTS users;

-- +goose StatementEnd
