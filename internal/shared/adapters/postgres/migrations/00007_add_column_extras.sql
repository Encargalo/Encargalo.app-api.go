-- +goose Up
-- +goose StatementBegin

-- Asegurar columnas necesarias
ALTER TABLE customers.address
ADD COLUMN IF NOT EXISTS type varchar,
ADD COLUMN IF NOT EXISTS extra JSONB;

-- Migrar todas las direcciones como Home
UPDATE customers.address
SET
    type = 'Home',
    extra = jsonb_build_object(
        'details', reference
    )
WHERE extra IS NULL;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

-- Revertir solo los datos migrados
UPDATE customers.address
SET
    type = NULL,
    extra = NULL
WHERE type = 'Home'
  AND extra ? 'details';

-- +goose StatementEnd
