-- +goose Up
CREATE TABLE IF NOT EXISTS orders.payment_methods (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type        VARCHAR     NOT NULL UNIQUE,
    name        VARCHAR     NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    fields      JSONB       NOT NULL DEFAULT '[]',
    qr_url      TEXT,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW()
);

INSERT INTO orders.payment_methods (type, name, description, fields, qr_url) VALUES
    ('EfectivoBs',
     'Efectivo en Bolívares',
     'Pago en efectivo en moneda venezolana (Bs)',
     '[]',
     NULL),
    ('EfectivoDolar',
     'Efectivo en Dólares',
     'Pago en efectivo en dólares americanos (USD)',
     '[]',
     NULL),
    ('PagoMovil',
     'Pago Móvil',
     'Pago electrónico interbancario venezolano',
     '[{"key":"nombre","label":"Nombre del titular","required":true},{"key":"cedula","label":"Número de cédula","required":true},{"key":"telefono","label":"Número de teléfono","required":true},{"key":"banco","label":"Nombre del banco","required":true}]',
     'https://res.cloudinary.com/dxmvyitlc/image/upload/v1780021364/WhatsApp_Image_2026-05-28_at_9.40.25_PM_hst1le.jpg');

-- +goose Down
DROP TABLE IF EXISTS orders.payment_methods;
