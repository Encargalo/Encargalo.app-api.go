-- +goose Up
CREATE SCHEMA IF NOT EXISTS marketing;

CREATE TABLE IF NOT EXISTS marketing.banners (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_url text NOT NULL,
    link_url text,
    position integer NOT NULL DEFAULT 0,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_banners_active_position ON marketing.banners (active, position);

-- +goose Down
DROP TABLE IF EXISTS marketing.banners;
DROP SCHEMA IF EXISTS marketing;
