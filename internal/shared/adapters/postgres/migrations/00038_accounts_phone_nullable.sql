-- +goose Up
-- El UNIQUE existente en phone ya permite múltiples NULL (Postgres no compara
-- NULLs entre sí para efectos de unicidad), así que basta con quitar NOT NULL
-- para permitir cuentas creadas por Google sin teléfono todavía.
ALTER TABLE users.accounts
    ALTER COLUMN phone DROP NOT NULL;

-- Cuentas antiguas guardan "sin email" como '' en vez de NULL; normalizamos
-- para que el índice único de abajo no las trate como si todas compartieran
-- el mismo email.
UPDATE users.accounts SET email = NULL WHERE email = '';

-- email no tenía ninguna restricción de unicidad; la necesitamos para poder
-- resolver de forma determinística a qué cuenta pertenece un login de Google.
CREATE UNIQUE INDEX accounts_email_unique
    ON users.accounts (LOWER(email))
    WHERE email IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users.accounts_email_unique;

ALTER TABLE users.accounts
    ALTER COLUMN phone SET NOT NULL;
