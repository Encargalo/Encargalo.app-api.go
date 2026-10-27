-- +goose Up
-- Las cotizaciones de delivery viven en Redis con TTL y se borran al
-- consumirse, así que una cotización que no convirtió se evapora. Esta tabla
-- es el registro durable para medir conversión por tramo de distancia y hora
-- (mirror de trips.fare_quotes, ver 00088_trip_fare_quotes.sql).
CREATE TABLE IF NOT EXISTS orders.fare_quotes
(
    quote_id             UUID             NOT NULL PRIMARY KEY,
    customer_id          UUID             NOT NULL REFERENCES users.accounts (id),
    shop_id              UUID             NOT NULL REFERENCES business.shops (id),
    shop_latitude        DOUBLE PRECISION NOT NULL,
    shop_longitude       DOUBLE PRECISION NOT NULL,
    delivery_latitude    DOUBLE PRECISION NOT NULL,
    delivery_longitude   DOUBLE PRECISION NOT NULL,
    distance_meters      DOUBLE PRECISION NOT NULL,
    fee_usd              NUMERIC(12, 2)   NOT NULL,
    fee_bs               NUMERIC(14, 2)   NOT NULL,
    delivery_subsidy_usd NUMERIC(12, 2)   NOT NULL DEFAULT 0,
    bcv_rate             NUMERIC(14, 4)   NOT NULL,
    fare_band            VARCHAR          NOT NULL,
    created_at           TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_fare_quotes_created_at ON orders.fare_quotes (created_at);
CREATE INDEX IF NOT EXISTS idx_order_fare_quotes_customer_id ON orders.fare_quotes (customer_id);

-- quote_id en la orden es el enlace de conversión: una fila de fare_quotes sin
-- orden que la referencie es una cotización que el cliente no pidió.
ALTER TABLE orders.orders
    ADD COLUMN IF NOT EXISTS quote_id UUID;

CREATE INDEX IF NOT EXISTS idx_orders_quote_id ON orders.orders (quote_id)
    WHERE quote_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS orders.idx_orders_quote_id;
ALTER TABLE orders.orders DROP COLUMN IF EXISTS quote_id;
DROP TABLE IF EXISTS orders.fare_quotes;
