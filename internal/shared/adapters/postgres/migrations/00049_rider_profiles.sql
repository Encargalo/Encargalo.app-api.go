-- +goose Up
CREATE SCHEMA IF NOT EXISTS riders;

CREATE TABLE IF NOT EXISTS riders.profiles (
    user_id              UUID PRIMARY KEY REFERENCES users.accounts (id),
    status               TEXT NOT NULL DEFAULT 'draft'
                         CHECK (status IN ('draft', 'active', 'suspended')),
    profile_photo_url    TEXT,
    vehicle_type         TEXT,
    vehicle_plate        TEXT,
    vehicle_brand        TEXT,
    vehicle_model        TEXT,
    vehicle_color        TEXT,
    id_photo_url         TEXT,
    carnet_photo_url     TEXT,
    license_photo_url    TEXT,
    circulation_card_url TEXT,
    vehicle_photo_1_url  TEXT,
    vehicle_photo_2_url  TEXT,
    vehicle_photo_3_url  TEXT,
    vehicle_photo_4_url  TEXT,
    notes                TEXT,
    last_contacted_at    TIMESTAMPTZ,
    activated_at         TIMESTAMPTZ,
    activated_by         UUID REFERENCES users.accounts (id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_riders_profiles_status_updated
    ON riders.profiles (status, updated_at ASC);

-- +goose Down
DROP INDEX IF EXISTS riders.idx_riders_profiles_status_updated;
DROP TABLE IF EXISTS riders.profiles;
DROP SCHEMA IF EXISTS riders;
