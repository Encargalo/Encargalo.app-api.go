-- +goose Up
ALTER TABLE shipments.shipments ADD COLUMN IF NOT EXISTS package_photo_url VARCHAR;

-- +goose Down
ALTER TABLE shipments.shipments DROP COLUMN IF EXISTS package_photo_url;
