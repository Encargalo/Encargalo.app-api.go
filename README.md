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

# Correr migraciones
goose -dir internal/shared/adapters/postgres/migrations postgres "<DSN>" up
```

### Debug con VS Code

El repo incluye la configuración de VS Code en `.vscode/`. Para correr el servidor en modo debug:

1. Abre el proyecto en VS Code con la extensión de Go instalada.
2. Ve a **Run and Debug** (`Ctrl+Shift+D`).
3. Selecciona **Debug Server** y presiona `F5`.

Antes de lanzar, VS Code regenera automáticamente los docs de Swagger. El servidor queda disponible en `http://localhost:3000`.

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
