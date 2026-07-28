-- +goose Up
-- La migración 00025_delivery_fee_formula.sql tenía un typo: dropeaba
-- "minimun_delivery" (mal escrito) en vez de "minimum_delivery" (nombre real
-- desde 00002_baseline_tables.sql). maximum_delivery y coverage sí se
-- limpiaron correctamente en su momento; esta columna quedó huérfana y sin
-- uso en el código desde entonces.
ALTER TABLE business.shops DROP COLUMN IF EXISTS minimum_delivery;

-- +goose Down
ALTER TABLE business.shops ADD COLUMN IF NOT EXISTS minimum_delivery INTEGER;
