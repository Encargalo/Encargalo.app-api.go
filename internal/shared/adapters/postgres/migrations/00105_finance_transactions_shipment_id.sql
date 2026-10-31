-- +goose Up
ALTER TABLE finance.transactions ADD COLUMN IF NOT EXISTS shipment_id UUID;

-- +goose Down
ALTER TABLE finance.transactions DROP COLUMN IF EXISTS shipment_id;
