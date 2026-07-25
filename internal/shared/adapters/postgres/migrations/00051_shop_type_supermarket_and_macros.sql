-- +goose Up
ALTER TABLE business.shops DROP CONSTRAINT IF EXISTS shops_type_check;
ALTER TABLE business.shops
    ADD CONSTRAINT shops_type_check
    CHECK ((type)::TEXT = ANY (ARRAY['Restaurant'::TEXT, 'Supermarket'::TEXT]));

INSERT INTO products.macro_categories (id, name, slug) VALUES
    ('ac000000-0000-0000-0000-000000000014', 'Abarrotes',     'abarrotes'),
    ('ac000000-0000-0000-0000-000000000015', 'Frescos',       'frescos'),
    ('ac000000-0000-0000-0000-000000000016', 'Lácteos',       'lacteos'),
    ('ac000000-0000-0000-0000-000000000017', 'Bebidas Súper', 'bebidas-super'),
    ('ac000000-0000-0000-0000-000000000018', 'Aseo',          'aseo'),
    ('ac000000-0000-0000-0000-000000000019', 'Otros Súper',   'otros-super');

-- +goose Down
DELETE FROM products.macro_categories WHERE id IN (
    'ac000000-0000-0000-0000-000000000014',
    'ac000000-0000-0000-0000-000000000015',
    'ac000000-0000-0000-0000-000000000016',
    'ac000000-0000-0000-0000-000000000017',
    'ac000000-0000-0000-0000-000000000018',
    'ac000000-0000-0000-0000-000000000019'
);

ALTER TABLE business.shops DROP CONSTRAINT IF EXISTS shops_type_check;
ALTER TABLE business.shops
    ADD CONSTRAINT shops_type_check
    CHECK ((type)::TEXT = 'Restaurant'::TEXT);
