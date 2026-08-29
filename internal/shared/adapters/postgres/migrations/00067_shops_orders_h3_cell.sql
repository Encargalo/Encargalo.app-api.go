-- +goose Up
-- +goose StatementBegin
ALTER TABLE business.shops ADD COLUMN IF NOT EXISTS h3_cell VARCHAR;
ALTER TABLE orders.orders ADD COLUMN IF NOT EXISTS h3_cell VARCHAR;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders.orders DROP COLUMN IF EXISTS h3_cell;
ALTER TABLE business.shops DROP COLUMN IF EXISTS h3_cell;
-- +goose StatementEnd
