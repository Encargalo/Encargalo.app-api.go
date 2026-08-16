-- +goose Up
ALTER TABLE finance.transactions
    ADD COLUMN IF NOT EXISTS staff_id  UUID REFERENCES users.accounts (id),
    ADD COLUMN IF NOT EXISTS notes     TEXT,
    ADD COLUMN IF NOT EXISTS proof_url TEXT;

-- +goose Down
ALTER TABLE finance.transactions
    DROP COLUMN IF EXISTS proof_url,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS staff_id;
