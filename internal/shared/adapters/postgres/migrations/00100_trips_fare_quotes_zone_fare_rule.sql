-- +goose Up
-- Congela qué zone_fare_rule produjo fare_usd, igual que fare_band/bcv_rate:
-- si la regla se edita después, la cotización ya emitida no debe cambiar de
-- precio. NULL = se usó el fallback global de tripfare.go. Sin ON DELETE
-- CASCADE a propósito: zone_fare_rules nunca se borra físicamente (soft
-- delete vía active=false), así que esta referencia siempre puede resolverse.
ALTER TABLE trips.fare_quotes
    ADD COLUMN IF NOT EXISTS zone_fare_rule_id UUID REFERENCES trips.zone_fare_rules (zone_id);

CREATE INDEX IF NOT EXISTS idx_fare_quotes_zone_fare_rule_id ON trips.fare_quotes (zone_fare_rule_id)
    WHERE zone_fare_rule_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS trips.idx_fare_quotes_zone_fare_rule_id;
ALTER TABLE trips.fare_quotes
    DROP COLUMN IF EXISTS zone_fare_rule_id;
