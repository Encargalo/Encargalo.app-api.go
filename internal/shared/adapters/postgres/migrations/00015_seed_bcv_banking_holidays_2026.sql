-- +goose Up
-- +goose StatementBegin

-- Feriados Nacionales y Bancarios 2026
-- Fuente: Calendario Bancario BCV 2026 (https://bcv.org.ve/bcv/calendario-bancario)
INSERT INTO bcv.banking_holidays (date) VALUES
    -- Enero
    ('2026-01-01'), -- Año Nuevo
    ('2026-01-12'), -- Día de Reyes (trasladado del 06/01)
    ('2026-01-19'), -- Día de la Divina Pastora (trasladado del 14/01)

    -- Febrero
    ('2026-02-16'), -- Carnaval
    ('2026-02-17'), -- Carnaval

    -- Marzo
    ('2026-03-19'), -- Día de San José

    -- Abril
    ('2026-04-02'), -- Semana Santa (Jueves Santo)
    ('2026-04-03'), -- Semana Santa (Viernes Santo)
    ('2026-04-19'), -- Movimiento Precursor de la Independencia

    -- Mayo
    ('2026-05-01'), -- Día del Trabajador
    ('2026-05-18'), -- Ascensión del Señor (trasladado del 14/05)

    -- Junio
    ('2026-06-08'), -- Corpus Christi (trasladado del 04/06)
    ('2026-06-13'), -- Día de San Antonio (sábado, no se ejecuta)
    ('2026-06-24'), -- Batalla de Carabobo
    ('2026-06-29'), -- Día de San Pedro y San Pablo

    -- Julio
    ('2026-07-05'), -- Día de la Independencia
    ('2026-07-24'), -- Natalicio del Libertador

    -- Agosto
    ('2026-08-15'), -- Asunción de Nuestra Señora (sábado, no se ejecuta)

    -- Septiembre
    ('2026-09-14'), -- Día de la Virgen de Coromoto (trasladado del 11/09)

    -- Octubre
    ('2026-10-12'), -- Día de la Resistencia Indígena
    ('2026-10-26'), -- Día de San José Gregorio Hernández

    -- Noviembre
    ('2026-11-01'), -- Día de Todos los Santos
    ('2026-11-23'), -- Día de la Virgen del Rosario de Chiquinquirá (trasladado del 18/11)

    -- Diciembre
    ('2026-12-14'), -- Día de la Inmaculada Concepción (trasladado del 08/12)
    ('2026-12-24'), -- Nochebuena
    ('2026-12-25'), -- Natividad de Nuestro Señor
    ('2026-12-31'); -- Fin de Año

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM bcv.banking_holidays WHERE EXTRACT(YEAR FROM date) = 2026;

-- +goose StatementEnd
