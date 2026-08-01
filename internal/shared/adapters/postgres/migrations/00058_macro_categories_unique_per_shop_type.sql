-- +goose Up
-- La restricción UNIQUE original (00020_macro_categories.sql) es global sobre
-- name/slug, pero la taxonomía real reutiliza el mismo nombre de macro en
-- distintos shop_type (p.ej. "Bebidas" existe para Restaurant, Supermarket y
-- Turbo). Se reescopa la unicidad a (name, shop_type) / (slug, shop_type).
ALTER TABLE products.macro_categories DROP CONSTRAINT IF EXISTS macro_categories_name_key;
ALTER TABLE products.macro_categories DROP CONSTRAINT IF EXISTS macro_categories_slug_key;
ALTER TABLE products.macro_categories ADD CONSTRAINT macro_categories_name_shop_type_key UNIQUE (name, shop_type);
ALTER TABLE products.macro_categories ADD CONSTRAINT macro_categories_slug_shop_type_key UNIQUE (slug, shop_type);

-- +goose Down
ALTER TABLE products.macro_categories DROP CONSTRAINT IF EXISTS macro_categories_name_shop_type_key;
ALTER TABLE products.macro_categories DROP CONSTRAINT IF EXISTS macro_categories_slug_shop_type_key;
ALTER TABLE products.macro_categories ADD CONSTRAINT macro_categories_name_key UNIQUE (name);
ALTER TABLE products.macro_categories ADD CONSTRAINT macro_categories_slug_key UNIQUE (slug);
