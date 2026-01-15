-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

-- SE VAN A AGREGAR TODAS LAS TABLAS EXISTENTES
-- EN ORDEN DE CREACION INICIAL.

CREATE TABLE IF NOT EXISTS owners.accounts
(
    id         UUID         DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    name       VARCHAR      NOT NULL,
    email      VARCHAR      NOT NULL UNIQUE,
    phone      VARCHAR      NOT NULL UNIQUE,
    created_at TIMESTAMP    DEFAULT NOW(),
    updated_at TIMESTAMP    DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS licenses.licenses
(
    id         UUID         DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    name       VARCHAR      NOT NULL,
    price      INTEGER      NOT NULL,
    created_at TIMESTAMP    DEFAULT NOW(),
    updated_at TIMESTAMP    DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business.shops
(
    id               UUID       DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    owner_id         UUID       NOT NULL REFERENCES owners.accounts (id),
    license_id       UUID       DEFAULT '6e02bcef-0372-4e80-ad1b-ed5d348dde0e'::UUID REFERENCES licenses.licenses (id),
    name             VARCHAR    NOT NULL,
    tag              VARCHAR,
    address          VARCHAR,
    home_phone       VARCHAR    NOT NULL,
    latitude         DOUBLE PRECISION,
    longitude        DOUBLE PRECISION,
    logo_image       VARCHAR,
    opened           BOOLEAN    DEFAULT FALSE,
    type             VARCHAR    NOT NULL CONSTRAINT shops_type_check CHECK ((type)::TEXT = 'Restaurant'::TEXT),
    score            REAL       DEFAULT 0,
    license_status   VARCHAR    DEFAULT 'active'::CHARACTER VARYING,
    created_at       TIMESTAMP  DEFAULT NOW(),
    updated_at       TIMESTAMP  DEFAULT NOW(),
    deleted_at       TIMESTAMP,
    banner           VARCHAR,
    coverage         REAL,
    maximum_delivery INTEGER,
    minimum_delivery INTEGER
);

CREATE TABLE IF NOT EXISTS products.categories
(
    id          UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id     UUID        REFERENCES business.shops (id) CONSTRAINT categories_shop_id_fkey1 REFERENCES business.shops (id),
    name        VARCHAR     NOT NULL,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products.items
(
    id              UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id         UUID        NOT NULL REFERENCES business.shops (id) CONSTRAINT items_shop_id_fkey1 REFERENCES business.shops (id),
    category_id     UUID        NOT NULL REFERENCES products.categories (id) CONSTRAINT items_category_id_fkey1 REFERENCES products.categories (id),
    name            VARCHAR     NOT NULL,
    price           INTEGER     NOT NULL,
    image           VARCHAR,
    description     VARCHAR     NOT NULL,
    is_available    BOOLEAN     DEFAULT FALSE,
    score           REAL        DEFAULT 0,
    created_at      TIMESTAMP   DEFAULT NOW(),
    updated_at      TIMESTAMP   DEFAULT NOW(),
    deleted_at      TIMESTAMP,
    has_flavors     BOOLEAN     DEFAULT FALSE NOT NULL,
    sold            INTEGER     DEFAULT 0
);

CREATE TABLE IF NOT EXISTS products.flavors
(
    id          UUID        DEFAULT GEN_RANDOM_UUID() NOT NULL PRIMARY KEY,
    shop_id     UUID        NOT NULL REFERENCES business.shops (id),
    category_id UUID        NOT NULL REFERENCES products.categories (id),
    name        TEXT        NOT NULL,
    description TEXT,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products.additions
(
    id          UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id     UUID        NOT NULL,
    name        VARCHAR     NOT NULL,
    price       INTEGER     NOT NULL,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products.categories_adiciones
(
    id          UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id     UUID        NOT NULL,
    category_id UUID        NOT NULL,
    addition_id UUID        NOT NULL,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products.items_flavors
(
    id          UUID        DEFAULT GEN_RANDOM_UUID() NOT NULL PRIMARY KEY,
    item_id     UUID        NOT NULL CONSTRAINT items_flavors_product_id_fkey REFERENCES products.items (id) ON DELETE CASCADE,
    flavor_id   UUID        NOT NULL REFERENCES products.flavors (id) ON DELETE CASCADE,
    created_at  TIMESTAMP   DEFAULT NOW(),
    CONSTRAINT items_flavors_product_id_flavor_id_key
        UNIQUE (item_id, flavor_id)
);

CREATE TABLE IF NOT EXISTS products.items_rules
(
    id              UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    item_id         UUID        NOT NULL CONSTRAINT items_rules_product_id_fkey REFERENCES products.items (id),
    rule_key        TEXT        NOT NULL,
    rule_value      INTEGER     NOT NULL,
    selector_type   TEXT        NOT NULL,
    created_at      TIMESTAMP   DEFAULT NOW(),
    updated_at      TIMESTAMP   DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE TABLE IF NOT EXISTS customers.accounts
(
    id                  UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    name                VARCHAR     NOT NULL,
    sur_name            VARCHAR     NOT NULL,
    phone               VARCHAR     NOT NULL,
    email               VARCHAR,
    password            VARCHAR     NOT NULL,
    birthday_date       DATE        NOT NULL,
    activation_status   VARCHAR     DEFAULT 'in progress'::CHARACTER VARYING,
    created_at          TIMESTAMP   DEFAULT NOW(),
    updated_at          TIMESTAMP   DEFAULT NOW(),
    deleted_at          TIMESTAMP
);

CREATE TABLE IF NOT EXISTS customers.activate_account
(
    id              UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    customer_id     UUID,
    activation_code VARCHAR     NOT NULL,
    created_at      TIMESTAMP   DEFAULT NOW(),
    updated_at      TIMESTAMP   DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE TABLE IF NOT EXISTS customers.address
(
    id          UUID                DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    customer_id UUID                NOT NULL,
    alias       VARCHAR             NOT NULL,
    address     TEXT                NOT NULL,
    reference   VARCHAR             NOT NULL,
    latitude    DOUBLE PRECISION    NOT NULL,
    longitude   DOUBLE PRECISION    NOT NULL,
    created_at  TIMESTAMP           DEFAULT NOW(),
    updated_at  TIMESTAMP           DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business.batches
(
    id         UUID         DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id    UUID         NOT NULL,
    batch      INTEGER,
    opened     BOOLEAN      DEFAULT TRUE,
    created_at TIMESTAMP    DEFAULT NOW(),
    close_at   TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders.orders
(
    id             UUID             NOT NULL PRIMARY KEY,
    shop_id        UUID             NOT NULL REFERENCES business.shops (id),
    customer_id    UUID             NOT NULL REFERENCES customers.accounts (id),
    address        VARCHAR          NOT NULL,
    latitude       DOUBLE PRECISION NOT NULL,
    longitude      DOUBLE PRECISION NOT NULL,
    delivery_fee   INTEGER          NOT NULL,
    total_price    INTEGER          NOT NULL,
    method_payment VARCHAR          CONSTRAINT orders_method_payment_check CHECK ((method_payment)::TEXT = ANY (ARRAY [('Efectivo'::CHARACTER VARYING)::TEXT, ('Nequi'::CHARACTER VARYING)::TEXT])),
    status         VARCHAR          DEFAULT 'Procesando'::CHARACTER VARYING,
    created_at     TIMESTAMP        DEFAULT NOW(),
    delivered_at   TIMESTAMP,
    canceled_at    TIMESTAMP,
    batch_id       UUID             REFERENCES business.batches (id),
    number         INTEGER
);

CREATE TABLE IF NOT EXISTS orders.order_items
(
    id          UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    item_id     UUID        NOT NULL REFERENCES products.items (id) CONSTRAINT order_items_item_id_fkey1 REFERENCES products.items (id),
    order_id    UUID        NOT NULL REFERENCES orders.orders CONSTRAINT order_items_order_id_fkey1 REFERENCES orders.orders (id),
    amount      INTEGER     NOT NULL,
    unit_price  INTEGER     NOT NULL,
    total_price INTEGER     NOT NULL,
    observation TEXT,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP,
    name        VARCHAR,
    flavors     JSONB
);

CREATE TABLE IF NOT EXISTS orders.additions_order
(
    id            UUID      DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    addition_id   UUID      NOT NULL REFERENCES products.additions (id),
    order_item_id UUID      NOT NULL REFERENCES orders.order_items (id),
    amount        INTEGER   NOT NULL,
    unit_price    INTEGER   NOT NULL,
    total_price   INTEGER   NOT NULL,
    created_at    TIMESTAMP DEFAULT NOW(),
    updated_at    TIMESTAMP DEFAULT NOW(),
    deleted_at    TIMESTAMP,
    name          VARCHAR
);

CREATE TABLE IF NOT EXISTS employees.accounts
(
    id         UUID         DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id    UUID         NOT NULL,
    name       VARCHAR      NOT NULL,
    email      VARCHAR      NOT NULL,
    phone      VARCHAR      NOT NULL,
    password   VARCHAR      NOT NULL,
    role       VARCHAR      NOT NULL,
    enabled    BOOLEAN      DEFAULT TRUE,
    created_by UUID         NOT NULL,
    created_at TIMESTAMP    DEFAULT NOW(),
    updated_at TIMESTAMP    DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business.week
(
    id  UUID    DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    day INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS business.schedules
(
    id      UUID    DEFAULT UUID_GENERATE_V4(),
    shop_id UUID    NOT NULL REFERENCES business.shops (id),
    day_id  UUID    NOT NULL REFERENCES business.week (id),
    open    TIME    NOT NULL,
    close   TIME    NOT NULL,
    enabled BOOLEAN DEFAULT TRUE NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
