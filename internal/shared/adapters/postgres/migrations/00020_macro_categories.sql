-- +goose Up
CREATE TABLE products.macro_categories (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT        NOT NULL UNIQUE,
    slug       TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE products.categories
    ADD COLUMN macro_category_id UUID
        REFERENCES products.macro_categories(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_macro_categories_name_fts
    ON products.macro_categories USING GIN (to_tsvector('spanish', name));

CREATE INDEX IF NOT EXISTS idx_macro_categories_name_trgm
    ON products.macro_categories USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_categories_macro_id
    ON products.categories (macro_category_id);

-- +goose Down
ALTER TABLE products.categories DROP COLUMN IF EXISTS macro_category_id;
DROP INDEX IF EXISTS idx_categories_macro_id;
DROP INDEX IF EXISTS idx_macro_categories_name_trgm;
DROP INDEX IF EXISTS idx_macro_categories_name_fts;
DROP TABLE IF EXISTS products.macro_categories;
