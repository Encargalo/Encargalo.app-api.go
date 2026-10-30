-- +goose Up
-- Primer caso real de zona de tarifa (issue #376): Parroquia Charallave
-- (OSM, relación 2461475), 23 celdas H3 resolución 7. Precios acordados con
-- negocio al BCV de referencia 888.8909 (2026-10-28); la fuente de verdad es
-- USD, la conversión a Bs sigue pasando por bcvRate en el momento de cotizar.
INSERT INTO trips.zone_fare_rules (
    zone_id, name, included_km,
    day_min_fare_usd, day_extra_rate_per_km_usd,
    night_min_fare_usd, night_extra_rate_per_km_usd,
    early_morning_min_fare_usd, early_morning_extra_rate_per_km_usd
) VALUES (
    'e221927c-c6b2-493a-b89d-eeae89a596bd', 'Charallave', 3,
    1.2375, 0.30,
    1.575, 0.50,
    1.80, 0.70
);

INSERT INTO trips.zone_fare_cells (h3_cell, zone_id) VALUES
    ('87674c486ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4b4ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4b5ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c416ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a0ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a1ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a2ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a3ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a4ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a5ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4a6ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4acffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c480ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4aeffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c481ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c410ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4b0ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4b1ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c484ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c412ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c485ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c413ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd'),
    ('87674c4b3ffffff', 'e221927c-c6b2-493a-b89d-eeae89a596bd');

-- +goose Down
DELETE FROM trips.zone_fare_cells WHERE zone_id = 'e221927c-c6b2-493a-b89d-eeae89a596bd';
DELETE FROM trips.zone_fare_rules WHERE zone_id = 'e221927c-c6b2-493a-b89d-eeae89a596bd';
