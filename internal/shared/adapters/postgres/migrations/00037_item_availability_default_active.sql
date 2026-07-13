-- +goose Up
ALTER TABLE products.items
    ALTER COLUMN availability_status SET DEFAULT 'active';

ALTER TABLE products.items
    ALTER COLUMN is_available SET DEFAULT TRUE;

-- +goose Down
ALTER TABLE products.items
    ALTER COLUMN availability_status SET DEFAULT 'hidden';

ALTER TABLE products.items
    ALTER COLUMN is_available SET DEFAULT FALSE;
