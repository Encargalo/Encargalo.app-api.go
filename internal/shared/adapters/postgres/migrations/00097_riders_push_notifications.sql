-- +goose Up
-- FK contra users.accounts, no riders.profiles: hay riders activos legacy sin
-- fila en profiles (ver IsActiveRider en riders/app/profile.go) — un FK contra
-- profiles les rompería el registro del device token con un 500.
CREATE TABLE riders.device_tokens (
    rider_id   UUID PRIMARY KEY REFERENCES users.accounts(id) ON DELETE CASCADE,
    fcm_token  TEXT NOT NULL,
    platform   TEXT NOT NULL CHECK (platform IN ('android', 'ios')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sin TTL, a diferencia de la presencia en Redis (45s): respaldo para pushear
-- a riders sin presencia viva por última ubicación conocida (ver #push-notifications).
CREATE TABLE riders.last_known_locations (
    rider_id   UUID PRIMARY KEY REFERENCES users.accounts(id) ON DELETE CASCADE,
    latitude   DOUBLE PRECISION NOT NULL,
    longitude  DOUBLE PRECISION NOT NULL,
    mode       TEXT NOT NULL CHECK (mode IN ('orders', 'trips')),
    vehicle    TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_riders_last_known_locations_lat_lng ON riders.last_known_locations (latitude, longitude);

-- +goose Down
DROP TABLE IF EXISTS riders.last_known_locations;
DROP TABLE IF EXISTS riders.device_tokens;
