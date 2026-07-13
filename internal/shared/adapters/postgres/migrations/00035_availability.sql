-- +goose Up

ALTER TABLE products.categories
    ADD COLUMN IF NOT EXISTS availability_status TEXT NOT NULL DEFAULT 'active'
        CHECK (availability_status IN ('active', 'hidden', 'out_of_stock'));

ALTER TABLE products.items
    ADD COLUMN IF NOT EXISTS availability_status TEXT NOT NULL DEFAULT 'hidden'
        CHECK (availability_status IN ('active', 'hidden', 'out_of_stock'));

UPDATE products.items
SET availability_status = CASE
    WHEN is_available IS TRUE THEN 'active'
    ELSE 'hidden'
END
WHERE TRUE;

-- +goose Down

ALTER TABLE products.categories DROP COLUMN IF EXISTS availability_status;
ALTER TABLE products.items DROP COLUMN IF EXISTS availability_status;
