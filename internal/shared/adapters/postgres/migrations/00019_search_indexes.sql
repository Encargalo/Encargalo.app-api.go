-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_shops_name_fts
    ON business.shops USING GIN (to_tsvector('spanish', name));

CREATE INDEX IF NOT EXISTS idx_shops_name_trgm
    ON business.shops USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_items_search_fts
    ON products.items USING GIN (to_tsvector('spanish', name || ' ' || COALESCE(description, '')));

CREATE INDEX IF NOT EXISTS idx_items_name_trgm
    ON products.items USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_categories_name_fts
    ON products.categories USING GIN (to_tsvector('spanish', name));

CREATE INDEX IF NOT EXISTS idx_categories_name_trgm
    ON products.categories USING GIN (name gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS idx_categories_name_trgm;
DROP INDEX IF EXISTS idx_categories_name_fts;
DROP INDEX IF EXISTS idx_items_name_trgm;
DROP INDEX IF EXISTS idx_items_search_fts;
DROP INDEX IF EXISTS idx_shops_name_trgm;
DROP INDEX IF EXISTS idx_shops_name_fts;
