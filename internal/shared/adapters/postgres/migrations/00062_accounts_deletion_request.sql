-- +goose Up
-- Soporta la eliminación diferida de cuentas: el usuario solicita eliminar
-- su cuenta, queda "pendiente" 30 días (constante en código,
-- models.AccountDeletionGracePeriod). Si vuelve a iniciar sesión en ese
-- lapso, la solicitud se cancela. Si no vuelve, un worker marca deleted_at
-- (soft delete ya soportado por Account) sin borrar ninguna otra columna.
ALTER TABLE users.accounts ADD COLUMN IF NOT EXISTS deletion_requested_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_deletion_requested_at
    ON users.accounts (deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users.idx_accounts_deletion_requested_at;
ALTER TABLE users.accounts DROP COLUMN IF EXISTS deletion_requested_at;
