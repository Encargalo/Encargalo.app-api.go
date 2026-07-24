-- +goose Up

ALTER TABLE products.categories
    ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{"flavors":[],"selector_type":"multi_select"}'::jsonb;

ALTER TABLE products.items
    ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{"flavors":[],"min_selection":0,"max_selection":1}'::jsonb;

-- Catálogo de sabores -> categories.details
UPDATE products.categories c
SET details = jsonb_build_object(
    'flavors', COALESCE(c.flavors, '[]'::jsonb),
    'selector_type', COALESCE(
        (
            SELECT elem->>'selector_type'
            FROM products.items i,
                 jsonb_array_elements(COALESCE(i.rules, '[]'::jsonb)) AS elem
            WHERE i.category_id = c.id
              AND elem->>'selector_type' IS NOT NULL
              AND elem->>'selector_type' <> ''
            LIMIT 1
        ),
        'multi_select'
    )
)
WHERE c.flavors IS NOT NULL
   OR c.details = '{"flavors":[],"selector_type":"multi_select"}'::jsonb;

UPDATE products.categories c
SET details = jsonb_build_object(
    'flavors', COALESCE(c.details->'flavors', '[]'::jsonb),
    'selector_type', COALESCE(c.details->>'selector_type', 'multi_select')
)
WHERE c.flavors IS NULL;

-- Sabores + límites por item -> items.details
UPDATE products.items i
SET details = jsonb_build_object(
    'flavors', COALESCE(i.flavors, '[]'::jsonb),
    'min_selection', COALESCE(
        (
            SELECT (elem->>'rule_value')::int
            FROM jsonb_array_elements(COALESCE(i.rules, '[]'::jsonb)) AS elem
            WHERE elem->>'rule_key' IN ('min_flavors', 'min_selection')
            LIMIT 1
        ),
        0
    ),
    'max_selection', COALESCE(
        (
            SELECT (elem->>'rule_value')::int
            FROM jsonb_array_elements(COALESCE(i.rules, '[]'::jsonb)) AS elem
            WHERE elem->>'rule_key' IN ('max_flavors', 'max_selection')
            LIMIT 1
        ),
        1
    )
)
WHERE i.flavors IS NOT NULL
   OR i.rules IS NOT NULL
   OR i.details = '{"flavors":[],"min_selection":0,"max_selection":1}'::jsonb;

ALTER TABLE products.categories DROP COLUMN IF EXISTS flavors;
ALTER TABLE products.items DROP COLUMN IF EXISTS flavors;
ALTER TABLE products.items DROP COLUMN IF EXISTS rules;

-- +goose Down

ALTER TABLE products.categories
    ADD COLUMN IF NOT EXISTS flavors JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE products.items
    ADD COLUMN IF NOT EXISTS flavors JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE products.items
    ADD COLUMN IF NOT EXISTS rules JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE products.categories
SET flavors = COALESCE(details->'flavors', '[]'::jsonb);

UPDATE products.items
SET flavors = COALESCE(details->'flavors', '[]'::jsonb),
    rules = '[]'::jsonb;

ALTER TABLE products.categories DROP COLUMN IF EXISTS details;
ALTER TABLE products.items DROP COLUMN IF EXISTS details;
