-- +goose Up
-- BuildCustomerRegisterModel guardaba '' en vez de NULL cuando el registro no
-- traía email (bug de código, corregido aparte) — accounts_email_unique
-- (migración 00038) trata '' como no-nulo, así que la segunda cuenta sin
-- email siempre chocaba contra la primera.
UPDATE users.accounts SET email = NULL WHERE email = '';

-- +goose Down
-- No hay forma de recuperar qué filas tenían '' en vez de NULL; no-op.
