-- +goose Up
ALTER TABLE analytics.dim_shop ADD COLUMN IF NOT EXISTS h3_cell TEXT;
ALTER TABLE analytics.cube_order ADD COLUMN IF NOT EXISTS h3_cell TEXT;

CREATE INDEX IF NOT EXISTS idx_dim_shop_h3_cell ON analytics.dim_shop (h3_cell) WHERE h3_cell IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cube_order_h3_cell ON analytics.cube_order (h3_cell) WHERE h3_cell IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS analytics.idx_cube_order_h3_cell;
DROP INDEX IF EXISTS analytics.idx_dim_shop_h3_cell;
ALTER TABLE analytics.cube_order DROP COLUMN IF EXISTS h3_cell;
ALTER TABLE analytics.dim_shop DROP COLUMN IF EXISTS h3_cell;
