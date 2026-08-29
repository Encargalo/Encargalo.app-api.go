-- +goose Up
-- shared_preload_libraries ya incluye pg_stat_statements (lo agrega solo el
-- wrapper.sh del template de Postgres de Railway en cada arranque si no
-- está), pero eso solo carga la librería — falta crear la extensión a nivel
-- de base de datos para que la vista sea consultable. Staging la tenía
-- creada a mano; esta migración la deja versionada para que también exista
-- en producción y en cualquier ambiente nuevo.
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- +goose Down
DROP EXTENSION IF EXISTS pg_stat_statements;
