# Encargalo App — API

[![CI](https://github.com/Encargalo/Encargalo.app-api.go/actions/workflows/integration.yaml/badge.svg)](https://github.com/Encargalo/Encargalo.app-api.go/actions/workflows/integration.yaml)
[![Release](https://img.shields.io/github/v/release/Encargalo/Encargalo.app-api.go)](https://github.com/Encargalo/Encargalo.app-api.go/releases)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![codecov](https://codecov.io/gh/Encargalo/Encargalo.app-api.go/branch/main/graph/badge.svg)](https://codecov.io/gh/Encargalo/Encargalo.app-api.go)

REST API del backend de Encargalo, construida en Go con arquitectura hexagonal.

## Requisitos

| Herramienta | Versión mínima |
|-------------|----------------|
| Go | 1.26 |
| Docker | cualquier versión reciente |
| PostgreSQL | 15+ |
| Redis | 7+ |

Herramientas opcionales pero recomendadas:

```bash
go install github.com/swaggo/swag/cmd/swag@latest   # generación de docs Swagger
go install github.com/vektra/mockery/v2@latest       # generación de mocks
go install github.com/pressly/goose/v3/cmd/goose@latest  # migraciones
```

## Instalación

```bash
git clone <url-del-repo>
cd Encargalo.app-api.go

# Instalar dependencias
go get ./...

# Instalar git hooks (recomendado)
make install-hooks
```

## Configuración

La aplicación lee su configuración desde variables de entorno. Crea un archivo `.env` en la raíz basándote en los grupos requeridos:

```
SERVER_PORT=3000

POSTGRES_HOST=...
POSTGRES_PORT=5432
POSTGRES_USER=...
POSTGRES_PASSWORD=...
POSTGRES_DB=...

REDIS_HOST=...
REDIS_PORT=6379

JWT_SECRET=...

# Ver internal/shared/config/config.go para el listado completo
```

## Ejecución

```bash
# Levantar dependencias (postgres + redis)
make up

# Correr migraciones OLTP
goose -dir internal/shared/adapters/postgres/migrations postgres "<DSN>" up

# Correr API
go run ./cmd/server

# Correr ETL (config mínima: SERVER/POSTGRES/OLAP_POSTGRES/ENVIRONMENT)
go run ./cmd/etl
```

Migraciones OLAP viven en `internal/shared/adapters/postgres/migrations_olap/` y se aplican al arrancar `cmd/etl`. Docs: [docs/etl-railway.md](docs/etl-railway.md), [docs/grafana-olap-queries.md](docs/grafana-olap-queries.md).

### Debug con VS Code

El repo incluye la configuración de VS Code en `.vscode/`. Para correr el servidor en modo debug:

1. Abre el proyecto en VS Code con la extensión de Go instalada.
2. Ve a **Run and Debug** (`Ctrl+Shift+D`).
3. Selecciona **Debug Server** y presiona `F5`.

Antes de lanzar, VS Code regenera automáticamente los docs de Swagger. El servidor queda disponible en `http://localhost:3000`.

## Sesión en desarrollo local (sin Traefik)

> **Estado actual:** `AuthMiddleware.Auth` todavía decodifica el JWT de la cookie directo (como siempre) — nada de lo de abajo es necesario hoy para desarrollar local. Esto documenta el flujo que pasa a ser obligatorio cuando se complete el corte a Traefik (`Auth` dejará de leer la cookie y solo confiará en `X-Encargalo-Session`); el endpoint `/internal/verify-session` ya está disponible desde ahora para ir probando ese flujo en paralelo.

Cuando el corte esté activo, el reverse proxy (Traefik) validará la sesión por cada request vía su middleware `ForwardAuth` y le pasará al backend el resultado en el header `X-Encargalo-Session`. En local **no corre Traefik**, así que hay que simular ese paso a mano:

```bash
# 1. Login normal (igual que siempre) — obtiene la cookie encargalo-session
curl -i -X POST http://localhost:3000/auth/sign-in/users \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+58...","password":"..."}' \
  -c cookies.txt

# 2. Llamar a mano el endpoint que llamaría Traefik, con esa cookie + el
#    secreto compartido (mismo valor que INTERNAL_GATEWAY_SECRET en tu .env).
#    Es GET, no POST: ForwardAuth siempre llama con GET sin importar el
#    método original de la request.
curl -i -X GET http://localhost:3000/internal/verify-session \
  -b cookies.txt \
  -H "X-Internal-Gateway-Secret: <valor de INTERNAL_GATEWAY_SECRET>"

# La respuesta trae el header X-Encargalo-Session con el JSON de la sesión.

# 3. Pegar ese JSON como header en las siguientes requests a rutas protegidas
curl http://localhost:3000/business/products \
  -H "X-Encargalo-Session: <json copiado del paso 2>" \
  -H "X-Internal-Gateway-Secret: <valor de INTERNAL_GATEWAY_SECRET>"
```

Sin ambos headers (`X-Encargalo-Session` y `X-Internal-Gateway-Secret`), cualquier ruta protegida por `Auth`/`Require` responde `401`, aunque la cookie de sesión sea válida — la cookie solo sirve ya para el paso 2. En Postman conviene automatizar los pasos 1-2 con un pre-request script que guarde el resultado en variables de colección.

## Documentación Swagger

Con el servidor corriendo, abre en el navegador:

```
http://localhost:3000/docs/index.html
```

Para regenerar la documentación después de modificar un handler:

```bash
swag init -g cmd/server/main.go --parseDependency --parseInternal
```

## Comandos útiles

```bash
# Compilar
go build -v ./...

# Correr todos los tests
go test ./...

# Generar mocks
mockery

# Bajar dependencias (postgres + redis)
make down
```
