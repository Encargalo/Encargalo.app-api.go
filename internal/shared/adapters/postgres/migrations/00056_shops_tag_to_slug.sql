-- +goose Up
ALTER TABLE business.shops RENAME COLUMN tag TO slug;

-- +goose Down
ALTER TABLE business.shops RENAME COLUMN slug TO tag;
