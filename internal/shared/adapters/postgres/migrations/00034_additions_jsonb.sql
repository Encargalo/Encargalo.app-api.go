-- +goose Up

-- Enriquecer categories.details con bloque de adiciones embebido.
UPDATE products.categories c
SET details = COALESCE(details, '{}'::jsonb) || jsonb_build_object(
    'additions', jsonb_build_object(
        'enabled', EXISTS (
            SELECT 1
            FROM products.categories_adiciones ca
            WHERE ca.category_id = c.id
              AND ca.deleted_at IS NULL
        ),
        'selector_type', 'multi_select',
        'min_selection', 0,
        'max_selection', GREATEST(
            COALESCE((
                SELECT COUNT(*)::int
                FROM products.categories_adiciones ca
                WHERE ca.category_id = c.id
                  AND ca.deleted_at IS NULL
            ), 0),
            1
        ),
        'items', COALESCE((
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', a.id,
                    'name', a.name,
                    'price', a.price,
                    'max_quantity', 1
                ) ORDER BY a.name
            )
            FROM products.categories_adiciones ca
            JOIN products.additions a ON a.id = ca.addition_id
            WHERE ca.category_id = c.id
              AND ca.deleted_at IS NULL
              AND a.deleted_at IS NULL
        ), '[]'::jsonb)
    )
)
WHERE TRUE;

DROP TABLE IF EXISTS products.categories_adiciones;

-- Las órdenes guardan addition_id como UUID histórico; ya no referencia catálogo en tabla.
ALTER TABLE orders.additions_order
    DROP CONSTRAINT IF EXISTS additions_order_addition_id_fkey;

DROP INDEX IF EXISTS products.idx_additions_shop_id;
DROP TABLE IF EXISTS products.additions;

-- +goose Down

CREATE TABLE IF NOT EXISTS products.additions
(
    id          UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id     UUID        NOT NULL,
    name        VARCHAR     NOT NULL,
    price       INTEGER     NOT NULL,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products.categories_adiciones
(
    id          UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    shop_id     UUID        NOT NULL,
    category_id UUID        NOT NULL,
    addition_id UUID        NOT NULL,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

UPDATE products.categories c
SET details = (details - 'additions')
WHERE details ? 'additions';

ALTER TABLE orders.additions_order
    ADD CONSTRAINT additions_order_addition_id_fkey
    FOREIGN KEY (addition_id) REFERENCES products.additions (id);
