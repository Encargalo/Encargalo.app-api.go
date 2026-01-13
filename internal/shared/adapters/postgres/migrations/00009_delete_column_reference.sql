-- +goose Up
-- +goose StatementBegin

ALTER TABLE customers.address
DROP COLUMN IF EXISTS reference;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE addresses
ADD COLUMN reference TEXT;

-- +goose StatementEnd
