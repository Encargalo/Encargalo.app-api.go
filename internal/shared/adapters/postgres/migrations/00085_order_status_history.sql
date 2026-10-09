-- +goose Up
-- Registro de transiciones de estado de cada pedido para el timeline del historial.
-- Se alimenta con un trigger para cubrir todos los caminos que cambian el estado
-- (local, rider, auto-rechazo, revisión de pago) sin tocar cada uno.
CREATE TABLE IF NOT EXISTS orders.order_status_history (
    id        BIGSERIAL PRIMARY KEY,
    order_id  UUID        NOT NULL REFERENCES orders.orders (id) ON DELETE CASCADE,
    status    TEXT        NOT NULL,
    at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_order_status_history_order_id
    ON orders.order_status_history (order_id, at);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION orders.record_order_status() RETURNS trigger AS $$
BEGIN
    INSERT INTO orders.order_status_history (order_id, status, at)
    VALUES (NEW.id, NEW.status, now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_order_status_insert ON orders.orders;
CREATE TRIGGER trg_order_status_insert
    AFTER INSERT ON orders.orders
    FOR EACH ROW EXECUTE FUNCTION orders.record_order_status();

DROP TRIGGER IF EXISTS trg_order_status_update ON orders.orders;
CREATE TRIGGER trg_order_status_update
    AFTER UPDATE OF status ON orders.orders
    FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION orders.record_order_status();

-- +goose Down
DROP TRIGGER IF EXISTS trg_order_status_update ON orders.orders;
DROP TRIGGER IF EXISTS trg_order_status_insert ON orders.orders;
DROP FUNCTION IF EXISTS orders.record_order_status();
DROP TABLE IF EXISTS orders.order_status_history;
