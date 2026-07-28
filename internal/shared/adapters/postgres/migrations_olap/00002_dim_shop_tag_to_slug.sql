-- +goose Up
ALTER TABLE analytics.dim_shop RENAME COLUMN tag TO slug;

-- +goose Down
ALTER TABLE analytics.dim_shop RENAME COLUMN slug TO tag;
