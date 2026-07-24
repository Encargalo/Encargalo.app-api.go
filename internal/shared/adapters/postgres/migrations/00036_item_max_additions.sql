-- +goose Up

-- Quitar bloque de adiciones del JSONB de categorías (el catálogo son items de macro Adiciones).
UPDATE products.categories
SET details = details - 'additions'
WHERE details ? 'additions';

-- Normalizar snapshots de adiciones en items: id/name/price/available + max fijo 5.
UPDATE products.items i
SET details = jsonb_set(
    jsonb_set(
        details,
        '{additions}',
        COALESCE((
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', elem->>'id',
                    'name', elem->>'name',
                    'price', COALESCE((elem->>'price')::int, 0),
                    'available', COALESCE((elem->>'available')::boolean, true)
                )
            )
            FROM jsonb_array_elements(COALESCE(details->'additions', '[]'::jsonb)) elem
        ), '[]'::jsonb),
        true
    ),
    '{max_additions_selection}',
    CASE
        WHEN jsonb_typeof(details->'additions') = 'array'
             AND jsonb_array_length(details->'additions') > 0
        THEN '5'::jsonb
        ELSE '0'::jsonb
    END,
    true
)
WHERE details ? 'additions'
   OR details ? 'max_additions_selection';

-- +goose Down

UPDATE products.categories
SET details = COALESCE(details, '{}'::jsonb) || jsonb_build_object(
    'additions', jsonb_build_object(
        'enabled', false,
        'selector_type', 'multi_select',
        'min_selection', 0,
        'max_selection', 0,
        'items', '[]'::jsonb
    )
)
WHERE NOT (details ? 'additions');
