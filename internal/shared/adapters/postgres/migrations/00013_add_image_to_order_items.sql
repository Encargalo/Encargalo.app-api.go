-- +goose Up
-- +goose StatementBegin

ALTER TABLE orders.order_items
ADD COLUMN image TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE orders.order_items
DROP COLUMN image;

-- +goose StatementEnd
