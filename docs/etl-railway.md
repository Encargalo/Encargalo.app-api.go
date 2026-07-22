# ETL / OLAP — despliegue Railway

## Servicios en el mismo repo GitHub

| Servicio Railway | Dockerfile | Rol |
|---|---|---|
| API | `Dockerfile` | `cmd/server` |
| ETL | `Dockerfile.etl` (`RAILWAY_DOCKERFILE_PATH=Dockerfile.etl`) | `cmd/etl` |
| Postgres OLTP | plugin | BD principal |
| Postgres OLAP | plugin nuevo | warehouse `analytics` |
| Grafana | imagen grafana/grafana | paneles |

## Variables del servicio ETL

Además de las vars que ya exige `config` (postgres OLTP, jwt, redis, etc.):

```
OLAP_POSTGRES_HOST=...
OLAP_POSTGRES_PORT=5432
OLAP_POSTGRES_USER=...
OLAP_POSTGRES_PASSWORD=...
OLAP_POSTGRES_DATABASE=...
# Opcional. Default: minuto 48 de cada hora.
# ETL_SCHEDULE=48 * * * *
# Opcional. Re-lectura tras watermark (horas). Default 48.
# ETL_LOOKBACK_HOURS=48
# Opcional. Backfill sin watermark (días). Default 1825 (~5 años).
# ETL_INITIAL_LOOKBACK_DAYS=1825
SERVER_PORT=3000
```

Migraciones OLAP se aplican al arrancar el ETL (`migrations_olap/`).
La sincronización de datos corre **solo por cron** (default `:48` de cada hora). Solo expone `GET /health`.

Queries de paneles: [grafana-olap-queries.md](./grafana-olap-queries.md).
