-- +goose Up
ALTER TABLE riders.profiles DROP CONSTRAINT IF EXISTS profiles_status_check;
ALTER TABLE riders.profiles ADD CONSTRAINT profiles_status_check
    CHECK (status IN ('draft', 'pending_review', 'changes_requested', 'rejected', 'active', 'suspended'));

ALTER TABLE riders.profiles
    ADD COLUMN IF NOT EXISTS submitted_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewed_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewed_by   UUID REFERENCES users.accounts (id),
    ADD COLUMN IF NOT EXISTS review_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_riders_profiles_pending_review
    ON riders.profiles (submitted_at ASC)
    WHERE status = 'pending_review';

-- +goose Down
DROP INDEX IF EXISTS riders.idx_riders_profiles_pending_review;

UPDATE riders.profiles
SET status = 'draft'
WHERE status IN ('pending_review', 'changes_requested', 'rejected');

ALTER TABLE riders.profiles
    DROP COLUMN IF EXISTS review_reason,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS submitted_at;

ALTER TABLE riders.profiles DROP CONSTRAINT IF EXISTS profiles_status_check;
ALTER TABLE riders.profiles ADD CONSTRAINT profiles_status_check
    CHECK (status IN ('draft', 'active', 'suspended'));
