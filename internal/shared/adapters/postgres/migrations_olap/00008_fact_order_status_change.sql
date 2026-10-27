-- +goose Up
-- Espejo crudo de orders.order_status_history (alimentada por trigger en OLTP,
-- cubre todos los caminos que cambian el estado de un pedido). id coincide con
-- el id de la fila origen: estable entre resyncs, nunca se reescribe una fila
-- ya insertada porque el historial es append-only por naturaleza.
CREATE TABLE IF NOT EXISTS analytics.fact_order_status_change (
    id        BIGINT      PRIMARY KEY,
    order_id  UUID        NOT NULL,
    status    TEXT        NOT NULL,
    at        TIMESTAMPTZ NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_fact_order_status_change_order_id
    ON analytics.fact_order_status_change (order_id, at);

-- +goose Down
DROP TABLE IF EXISTS analytics.fact_order_status_change;
