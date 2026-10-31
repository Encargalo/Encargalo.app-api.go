-- +goose Up
-- Especialidad del negocio (Panadería, Carnicería, etc.), independiente del
-- type (modelo de negocio: Restaurant/Supermarket/Turbo). Un Supermarket
-- puede ser en realidad solo una panadería. NULL = sin especialidad
-- declarada (tratado como "General" en el código, no requiere backfill).
ALTER TABLE business.shops ADD COLUMN IF NOT EXISTS specialty VARCHAR NULL;

-- +goose Down
ALTER TABLE business.shops DROP COLUMN IF EXISTS specialty;
