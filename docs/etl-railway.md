# ETL / OLAP — despliegue Railway

## Servicios en el mismo repo GitHub

| Servicio Railway | Dockerfile | Rol |
|---|---|---|
| API | `Dockerfile` | `cmd/server` |
| ETL | `Dockerfile.etl` (`RAILWAY_DOCKERFILE_PATH=Dockerfile.etl`) | `cmd/etl` |
| Postgres OLTP | plugin | BD principal |
| Postgres OLAP | plugin nuevo | warehouse `analytics` |
| Grafana | imagen grafana/grafana | paneles |

## Variables del servicio ETL (config mínima)

El ETL **no** usa JWT/Redis/Google/Cloudinary. Solo:

```
ENVIRONMENT_CURRENT=production

POSTGRES_HOST=${{Postgres.PGHOST}}
POSTGRES_PORT=${{Postgres.PGPORT}}
POSTGRES_USER=${{Postgres.PGUSER}}
POSTGRES_PASSWORD=${{Postgres.PGPASSWORD}}
POSTGRES_DATABASE=${{Postgres.PGDATABASE}}

OLAP_POSTGRES_HOST=${{PostgresOLAP.PGHOST}}
OLAP_POSTGRES_PORT=${{PostgresOLAP.PGPORT}}
OLAP_POSTGRES_USER=${{PostgresOLAP.PGUSER}}
OLAP_POSTGRES_PASSWORD=${{PostgresOLAP.PGPASSWORD}}
OLAP_POSTGRES_DATABASE=${{PostgresOLAP.PGDATABASE}}

RAILWAY_DOCKERFILE_PATH=Dockerfile.etl

# SERVER_PORT es opcional: si no está, se usa el PORT que inyecta Railway.
# SERVER_PORT=${{PORT}}

# Opcionales
# ETL_SCHEDULE=48 * * * *
# ETL_LOOKBACK_HOURS=48
# ETL_INITIAL_LOOKBACK_DAYS=1825
```

Migraciones OLAP se aplican al arrancar el ETL (`migrations_olap/`).
Sync solo por cron (default `:48` de cada hora). Expone `GET /health`.

Queries de paneles: [grafana-olap-queries.md](./grafana-olap-queries.md).
