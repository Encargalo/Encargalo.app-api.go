-- +goose Up

-- Antes de crear el índice único, se eliminan duplicados existentes
-- (misma categoría + misma adición) dejando solo la fila más reciente, para
-- que la migración no falle si hay datos sucios de antes del CRUD.
DELETE FROM products.categories_adiciones a
    USING products.categories_adiciones b
WHERE a.category_id = b.category_id
  AND a.addition_id = b.addition_id
  AND a.deleted_at IS NULL
  AND b.deleted_at IS NULL
  AND a.created_at < b.created_at;

-- Evita que una misma adición quede vinculada dos veces a la misma
-- categoría (una adición sí puede repetirse en categorías distintas).
CREATE UNIQUE INDEX IF NOT EXISTS ux_categories_adiciones_category_addition
    ON products.categories_adiciones (category_id, addition_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_categories_adiciones_addition_id
    ON products.categories_adiciones (addition_id);

CREATE INDEX IF NOT EXISTS idx_additions_shop_id
    ON products.additions (shop_id);

-- +goose Down
DROP INDEX IF EXISTS idx_additions_shop_id;
DROP INDEX IF EXISTS idx_categories_adiciones_addition_id;
DROP INDEX IF EXISTS ux_categories_adiciones_category_addition;
