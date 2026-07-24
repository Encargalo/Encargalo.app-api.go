-- +goose Up

ALTER TABLE products.categories
    ADD COLUMN IF NOT EXISTS flavors JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE products.items
    ADD COLUMN IF NOT EXISTS flavors JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE products.items
    ADD COLUMN IF NOT EXISTS rules JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Catálogo de sabores por categoría (desde products.flavors).
UPDATE products.categories c
SET flavors = COALESCE(
    (
        SELECT jsonb_agg(
            jsonb_build_object(
                'id', f.id,
                'name', f.name,
                'description', COALESCE(f.description, '')
            ) ORDER BY f.created_at
        )
        FROM products.flavors f
        WHERE f.category_id = c.id
          AND f.deleted_at IS NULL
    ),
    '[]'::jsonb
);

-- Sabores asignados por item (desde products.items_flavors).
UPDATE products.items i
SET flavors = COALESCE(
    (
        SELECT jsonb_agg(
            jsonb_build_object(
                'id', f.id,
                'name', f.name,
                'description', COALESCE(f.description, '')
            )
        )
        FROM products.items_flavors ifl
        JOIN products.flavors f ON f.id = ifl.flavor_id
        WHERE ifl.item_id = i.id
          AND f.deleted_at IS NULL
    ),
    '[]'::jsonb
),
has_flavors = EXISTS (
    SELECT 1
    FROM products.items_flavors ifl
    WHERE ifl.item_id = i.id
);

-- Reglas por item (desde products.items_rules).
UPDATE products.items i
SET rules = COALESCE(
    (
        SELECT jsonb_agg(
            jsonb_build_object(
                'id', r.id,
                'rule_key', r.rule_key,
                'rule_value', r.rule_value,
                'selector_type', r.selector_type
            ) ORDER BY r.created_at
        )
        FROM products.items_rules r
        WHERE r.item_id = i.id
          AND r.deleted_at IS NULL
    ),
    '[]'::jsonb
);

DROP TABLE IF EXISTS products.items_flavors;
DROP TABLE IF EXISTS products.items_rules;
DROP TABLE IF EXISTS products.flavors;

-- +goose Down

CREATE TABLE IF NOT EXISTS products.flavors
(
    id          UUID        DEFAULT GEN_RANDOM_UUID() NOT NULL PRIMARY KEY,
    shop_id     UUID        NOT NULL REFERENCES business.shops (id),
    category_id UUID        NOT NULL REFERENCES products.categories (id),
    name        TEXT        NOT NULL,
    description TEXT,
    created_at  TIMESTAMP   DEFAULT NOW(),
    updated_at  TIMESTAMP   DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products.items_flavors
(
    id          UUID        DEFAULT GEN_RANDOM_UUID() NOT NULL PRIMARY KEY,
    item_id     UUID        NOT NULL REFERENCES products.items (id) ON DELETE CASCADE,
    flavor_id   UUID        NOT NULL REFERENCES products.flavors (id) ON DELETE CASCADE,
    created_at  TIMESTAMP   DEFAULT NOW(),
    CONSTRAINT items_flavors_product_id_flavor_id_key UNIQUE (item_id, flavor_id)
);

CREATE TABLE IF NOT EXISTS products.items_rules
(
    id              UUID        DEFAULT UUID_GENERATE_V4() NOT NULL PRIMARY KEY,
    item_id         UUID        NOT NULL REFERENCES products.items (id),
    rule_key        TEXT        NOT NULL,
    rule_value      INTEGER     NOT NULL,
    selector_type   TEXT        NOT NULL,
    created_at      TIMESTAMP   DEFAULT NOW(),
    updated_at      TIMESTAMP   DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

ALTER TABLE products.categories DROP COLUMN IF EXISTS flavors;
ALTER TABLE products.items DROP COLUMN IF EXISTS flavors;
ALTER TABLE products.items DROP COLUMN IF EXISTS rules;
