-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

ALTER TABLE products.items
ADD COLUMN image_public_id TEXT,
ADD COLUMN image_hash CHAR(64);


-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
