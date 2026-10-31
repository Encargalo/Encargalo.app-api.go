-- +goose Up
ALTER TABLE business.week ADD CONSTRAINT uq_week_day UNIQUE (day);
INSERT INTO business.week (day) VALUES (0),(1),(2),(3),(4),(5),(6)
ON CONFLICT (day) DO NOTHING;

-- +goose Down
DELETE FROM business.week WHERE day BETWEEN 0 AND 6;
ALTER TABLE business.week DROP CONSTRAINT IF EXISTS uq_week_day;
