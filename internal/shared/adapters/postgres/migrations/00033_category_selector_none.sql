-- +goose Up

ALTER TABLE products.categories
    ALTER COLUMN details SET DEFAULT '{"flavors":[],"selector_type":"none"}'::jsonb;

-- Categorías sin catálogo de sabores pasan a none si quedaron con multi_select por defecto.
UPDATE products.categories
SET details = jsonb_set(details, '{selector_type}', '"none"'::jsonb, true)
WHERE COALESCE(jsonb_array_length(details->'flavors'), 0) = 0
  AND COALESCE(details->>'selector_type', 'multi_select') = 'multi_select';

-- +goose Down

ALTER TABLE products.categories
    ALTER COLUMN details SET DEFAULT '{"flavors":[],"selector_type":"multi_select"}'::jsonb;
