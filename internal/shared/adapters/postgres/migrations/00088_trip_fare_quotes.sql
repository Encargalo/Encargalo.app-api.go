-- +goose Up
-- Las cotizaciones viven en Redis con TTL de 10 min y se borran al consumirse
-- (GETDEL), así que una cotización que no convirtió se evapora. Esta tabla es
-- el registro durable para medir conversión por tramo de distancia y de hora.
CREATE TABLE IF NOT EXISTS trips.fare_quotes
(
    quote_id            UUID             NOT NULL PRIMARY KEY,
    customer_id         UUID             NOT NULL REFERENCES users.accounts (id),
    pickup_latitude     DOUBLE PRECISION NOT NULL,
    pickup_longitude    DOUBLE PRECISION NOT NULL,
    dropoff_latitude    DOUBLE PRECISION NOT NULL,
    dropoff_longitude   DOUBLE PRECISION NOT NULL,
    pickup_h3_cell_zone VARCHAR,
    dropoff_h3_cell_zone VARCHAR,
    distance_meters     DOUBLE PRECISION NOT NULL,
    duration_min        BIGINT           NOT NULL,
    fare_usd            NUMERIC(12, 2)   NOT NULL,
    fare_bs             NUMERIC(14, 2)   NOT NULL,
    bcv_rate            NUMERIC(14, 4)   NOT NULL,
    fare_band           VARCHAR          NOT NULL,
    created_at          TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fare_quotes_created_at ON trips.fare_quotes (created_at);
CREATE INDEX IF NOT EXISTS idx_fare_quotes_customer_id ON trips.fare_quotes (customer_id);

-- quote_id en el trip es el enlace de conversión: una fila de fare_quotes sin
-- trip que la referencie es una cotización que el cliente no pidió.
ALTER TABLE trips.trips
    ADD COLUMN IF NOT EXISTS quote_id UUID;

CREATE INDEX IF NOT EXISTS idx_trips_quote_id ON trips.trips (quote_id)
    WHERE quote_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS trips.idx_trips_quote_id;
ALTER TABLE trips.trips DROP COLUMN IF EXISTS quote_id;
DROP TABLE IF EXISTS trips.fare_quotes;
