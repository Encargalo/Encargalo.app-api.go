---
name: contexto-proyecto
description: Contexto completo del proyecto Encargalo.app-api.go — módulos, stack, arquitectura, lenguaje ubicuo y convenciones.
---

# Contexto del Proyecto — Encargalo.app-api.go

## Identidad del sistema
- **Nombre**: Encargalo.app-api.go
- **Tipo**: API REST backend
- **Lenguaje**: Go 1.26
- **Framework HTTP**: Echo v4
- **Arquitectura**: Hexagonal (Ports & Adapters)
- **ORM**: bun v1 + goose (migraciones)
- **Cache / Streams**: Redis (redsumer, Redis Streams)
- **DI**: uber/dig
- **Auth**: JWT en cookie `encargalo-session`
- **WebSocket**: gorilla/websocket (gestión de órdenes en tiempo real)
- **Storage**: Cloudinary (imágenes)
- **Notificaciones**: Discord

## Estructura de carpetas

```
cmd/server/             — punto de entrada + DI (providers/di.go)
api/router/             — registro de rutas (router.go + groups/)
internal/<dominio>/
  domain/
    models/             — entidades de dominio
    ports/              — interfaces (XxxApp + XxxRepo)
    dtos/               — DTOs request/response
    entity/             — value objects / entidades auxiliares
  handler/              — HTTP handlers (Echo)
  app/                  — lógica de negocio (implementa ports.XxxApp)
  repo/                 — acceso a datos (implementa ports.XxxRepo)
internal/shared/
  config/               — variables de entorno (gosuite)
  adapters/postgres/    — conexión bun + migraciones goose
  adapters/redis/       — cliente Redis + stream producer
  jwt/                  — sign/validate JWT
  errcustom/            — errores centinela
internal/pkg/           — utilidades: bycript, cache, cookie, json, logs, uploadImage
internal/mocks/         — mocks generados por mockery
internal/test/          — tests de handlers y casos de uso
docs/                   — Swagger generado (swag)
postman/                — colecciones Postman por dominio
specs/                  — especificaciones técnicas
```

## Módulos / Bounded Contexts

| Módulo | Ruta | Propósito |
|---|---|---|
| `auth` | `internal/auth/` | Autenticación, sesiones, roles |
| `users` | `internal/users/` | Gestión de usuarios/clientes |
| `shops` | `internal/shops/` | Tiendas y búsqueda |
| `products` | `internal/products/` | Productos del catálogo |
| `orders` | `internal/orders/` | Ciclo de vida de pedidos |
| `employee` | `internal/employee/` | Empleados de tienda |
| `payment_methods` | `internal/payment_methods/` | Métodos de pago |
| `bcv` | `internal/bcv/` | Tasa de cambio BCV |
| `health` | `internal/health/` | Health check |

## Roles del sistema
Definidos en `internal/auth/domain/models/role.go`:
- `Customer` 
- `OrderManager`
- `Owner`

## Lenguaje ubicuo

| Término | Significado |
|---|---|
| Shop | Tienda registrada en la plataforma |
| Batch | Lote de productos de una tienda en un pedido |
| Order | Pedido del cliente (contiene uno o más Batches) |
| Employee | Trabajador asignado a una tienda |
| OrderManager | Rol que gestiona pedidos en tiempo real vía WebSocket |
| Owner | Propietario de tienda |
| BCV | Banco Central de Venezuela — fuente de la tasa de cambio |
| `On The Way` | Estado de orden que activa el cobro de comisión (10%) |
| CreditManager | Servicio externo que recibe la comisión de cada orden |

## Convenciones de naming

- Interfaces en ports: `XxxApp`, `XxxRepo` (e.g. `ShopsApp`, `ShopsRepo`)
- Handlers: `XxxHandler` — struct en `handler/<modulo>.go`
- Apps (casos de uso): struct en `app/<modulo>.go` que implementa `XxxApp`
- Repos: struct en `repo/<modulo>.go` que implementa `XxxRepo`
- DTOs: nombres descriptivos en `dtos/` (e.g. `SearchShopsQuery`, `ShopsResponse`)
- Modelos: en `domain/models/` o `domain/models/<entidad>/`
- Tests: en `internal/test/<dominio>/`
- Mocks: en `internal/mocks/` — generados con mockery

## Integración externa
- **CreditManager**: servicio HTTP externo — recibe 10% de `totalPrice - deliveryFee` cuando una orden alcanza `On The Way`
- **Redis Streams**: eventos de órdenes publicados en `ports.RedisStream.PublisherEvent`
- **Cloudinary**: almacenamiento de imágenes de productos y tiendas

## Variables de entorno requeridas
Grupos: `server`, `postgres`, `redis`, `redis_stream`, `jwt`, `environment`, `google`, `cloudinary`, `credit_manager`, `discord`.
Config en: `internal/shared/config/config.go`.
