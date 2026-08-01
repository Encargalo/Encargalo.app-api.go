-- +goose Up
-- Catálogo global de micro-categorías (subcategorías reales dentro de una
-- macro-categoría), fuente para SeedDefaultCategories: el onboarding de
-- shops por staff crea una products.categories por cada micro-categoría del
-- shop_type correspondiente, ya asociada a su macro vía macro_category_id.
CREATE TABLE products.micro_categories (
    id                 UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    macro_category_id  UUID        NOT NULL REFERENCES products.macro_categories(id) ON DELETE CASCADE,
    name               TEXT        NOT NULL,
    slug               TEXT        NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (macro_category_id, slug)
);

CREATE INDEX idx_micro_categories_macro_category_id ON products.micro_categories (macro_category_id);

-- +goose Down
DROP INDEX IF EXISTS products.idx_micro_categories_macro_category_id;
DROP TABLE IF EXISTS products.micro_categories;
