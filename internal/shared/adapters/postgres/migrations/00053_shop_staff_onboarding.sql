-- +goose Up
ALTER TABLE business.shops ADD COLUMN status varchar DEFAULT 'draft' NOT NULL;
ALTER TABLE business.shops ADD COLUMN activated_at timestamp;
ALTER TABLE business.shops ADD COLUMN activated_by uuid;
ALTER TABLE business.shops ALTER COLUMN home_phone DROP NOT NULL;

-- Backfill: todo shop ya existente antes de este change queda operativo.
UPDATE business.shops SET status = 'active';

-- +goose Down
ALTER TABLE business.shops ALTER COLUMN home_phone SET NOT NULL;
ALTER TABLE business.shops DROP COLUMN activated_by;
ALTER TABLE business.shops DROP COLUMN activated_at;
ALTER TABLE business.shops DROP COLUMN status;
