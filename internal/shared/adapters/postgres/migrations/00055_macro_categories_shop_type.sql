-- +goose Up
ALTER TABLE products.macro_categories ADD COLUMN shop_type VARCHAR;

UPDATE products.macro_categories SET shop_type = 'Restaurant' WHERE id IN (
    'ac000000-0000-0000-0000-000000000001',
    'ac000000-0000-0000-0000-000000000002',
    'ac000000-0000-0000-0000-000000000003',
    'ac000000-0000-0000-0000-000000000004',
    'ac000000-0000-0000-0000-000000000005',
    'ac000000-0000-0000-0000-000000000006',
    'ac000000-0000-0000-0000-000000000007',
    'ac000000-0000-0000-0000-000000000008',
    'ac000000-0000-0000-0000-000000000009',
    'ac000000-0000-0000-0000-000000000010',
    'ac000000-0000-0000-0000-000000000011',
    'ac000000-0000-0000-0000-000000000012',
    'ac000000-0000-0000-0000-000000000013'
);

UPDATE products.macro_categories SET shop_type = 'Supermarket' WHERE id IN (
    'ac000000-0000-0000-0000-000000000014',
    'ac000000-0000-0000-0000-000000000015',
    'ac000000-0000-0000-0000-000000000016',
    'ac000000-0000-0000-0000-000000000017',
    'ac000000-0000-0000-0000-000000000018',
    'ac000000-0000-0000-0000-000000000019'
);

INSERT INTO products.macro_categories (id, name, slug, shop_type) VALUES
    ('ac000000-0000-0000-0000-000000000020', 'Snacks',               'snacks',              'Turbo'),
    ('ac000000-0000-0000-0000-000000000021', 'Bebidas Turbo',        'bebidas-turbo',       'Turbo'),
    ('ac000000-0000-0000-0000-000000000022', 'Dulces',               'dulces',              'Turbo'),
    ('ac000000-0000-0000-0000-000000000023', 'Licores y Cigarrillos','licores-cigarrillos', 'Turbo'),
    ('ac000000-0000-0000-0000-000000000024', 'Farmacia Básica',      'farmacia-basica',     'Turbo'),
    ('ac000000-0000-0000-0000-000000000025', 'Aseo Personal',        'aseo-personal',       'Turbo'),
    ('ac000000-0000-0000-0000-000000000026', 'Otros Turbo',          'otros-turbo',         'Turbo');

ALTER TABLE products.macro_categories ALTER COLUMN shop_type SET NOT NULL;
ALTER TABLE products.macro_categories
    ADD CONSTRAINT macro_categories_shop_type_check
    CHECK (shop_type IN ('Restaurant', 'Supermarket', 'Turbo'));

CREATE INDEX idx_macro_categories_shop_type ON products.macro_categories (shop_type);

-- +goose Down
DROP INDEX IF EXISTS products.idx_macro_categories_shop_type;
ALTER TABLE products.macro_categories DROP CONSTRAINT IF EXISTS macro_categories_shop_type_check;
DELETE FROM products.macro_categories WHERE id IN (
    'ac000000-0000-0000-0000-000000000020',
    'ac000000-0000-0000-0000-000000000021',
    'ac000000-0000-0000-0000-000000000022',
    'ac000000-0000-0000-0000-000000000023',
    'ac000000-0000-0000-0000-000000000024',
    'ac000000-0000-0000-0000-000000000025',
    'ac000000-0000-0000-0000-000000000026'
);
ALTER TABLE products.macro_categories DROP COLUMN IF EXISTS shop_type;
