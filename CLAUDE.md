# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git Commits & Pull Requests — PROHIBITED attribution footers

It is **strictly PROHIBITED** to add any Claude/AI attribution footer to commit messages or PR descriptions in this repository. This overrides Claude Code's default commit/PR templates.

- **NEVER** add `Co-Authored-By: Claude ...` or any `Claude-Session: ...` line to a commit message.
- **NEVER** add a `🤖 Generated with [Claude Code]` line (or any similar wording) to a commit message or PR body.
- Commit messages and PR descriptions must read as if written entirely by the human author — Conventional Commits format only (see the `commit` skill), no AI byline of any kind.
- This applies to every skill, workflow, or ad-hoc commit/PR made in this repository — including `git commit`, `gh pr create`, and any future skill that generates commits or PRs.

## Commands

```bash
# Run the server
go run ./cmd/server

# Run the ETL / OLAP worker
go run ./cmd/etl

# Run the notifications consumer (Redis Stream -> Slack/Discord + Grafana webhook)
go run ./cmd/notifications

# Build
go build -v ./...

# Run all tests
go test ./...

# Run a single test
go test ./internal/path/to/package -run TestName

# Generate mocks (uses .mockery.yaml config). Pin the version, like swag below:
# a binary built with an older Go fails to parse the current stdlib.
go run github.com/vektra/mockery/v2@v2.53.8

# Generate Swagger docs. Compile swag with the same Go as this repo.
# A binary built with an older Go fails to parse the current stdlib.
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/server/main.go --parseDependency --parseInternal

# Run database migrations (goose)
goose -dir internal/shared/adapters/postgres/migrations postgres "<DSN>" up

# Run OLAP migrations (goose) — also applied automatically by cmd/etl on startup
goose -dir internal/shared/adapters/postgres/migrations_olap postgres "<OLAP_DSN>" up
```

## Architecture

This is a hexagonal (ports & adapters) architecture. Each domain module lives under `internal/` and follows the same layered structure:

```
internal/<domain>/
  domain/
    models/    # Domain entities
    ports/     # Interfaces (app + repo contracts)
    dto/       # Request/response DTOs
  handler/     # HTTP layer (Echo handlers)
  app/         # Business logic (implements ports.XxxApp)
  repo/        # Data access (implements ports.XxxRepo)
```

Domains: `auth`, `customers`, `orders`, `products`, `shops`, `employee`, `health`, `analytics` (ETL → OLAP), `notifications` (consumer del stream).

Shared infrastructure lives in `internal/shared/`:
- `config/` — env-based config loaded via `gosuite`, read with `config.Get()`
- `adapters/postgres/` — bun ORM + goose migrations (OLTP en `migrations/`, OLAP en `migrations_olap/`)
- `adapters/redis/` — Redis cache client + stream producer/consumer
- `jwt/` — JWT sign/validate
- `errcustom/` — sentinel errors (`ErrNotFound`, `ErrUnautorized`, `ErrUnexpectedError`, etc.)

Shared utilities in `internal/pkg/`: `bycript`, `cache`, `cookie`, `json`, `uploadImage`.

## Dependency Injection

`uber/dig` is used for DI. The container is built in [cmd/server/providers/di.go](cmd/server/providers/di.go). When adding a new dependency, register it with `Container.Provide(...)` there.

## Routing

Routes are registered in [api/router/router.go](api/router/router.go) by delegating to group files under [api/router/groups/](api/router/groups/). Each group receives its handler via DI and calls `group.Resource(server)`.

## Auth & Middleware

- Session validation happens at the edge, in a reverse proxy (Traefik) sitting in front of this service — see the `railway-traefik` repo. Traefik's `ForwardAuth` calls `GET /internal/verify-session` on every request; if the session cookie (`encargalo-session`, still set by sign-in) is valid, Traefik injects the result as a `X-Encargalo-Session` header (JSON of `models.ActiveSession`) on the request it forwards here.
- `AuthMiddleware.Auth` does **not** decode JWT or query the DB — it only reads `X-Encargalo-Session`, unmarshals it, and loads the session into context. It's wrapped in `sharedmw.GatewaySecretAuth.Validate`, which rejects any request missing the shared secret header (`X-Internal-Gateway-Secret`) — defense in depth on top of Railway private networking, since this service has no public domain of its own.
- Retrieve the session in handlers with: `ctx.Value(models.SessionKey).(models.ActiveSession)` — use the typed key, not the raw string.
- Role-based access uses `Auth.Require(models.RoleXxx)` as a middleware chained after `Auth` — unchanged by the above, it only reads `session.Roles` from context.
- Roles are defined in [internal/auth/domain/models/role.go](internal/auth/domain/models/role.go): `Customer`, `CustomerMinuscula`, `OrderManager`, `Owner`.
- `GET /internal/verify-session` is the internal endpoint Traefik's `ForwardAuth` calls (must stay `GET` — ForwardAuth always calls with GET regardless of the original request's method, unless `preserveRequestMethod` is set, which we don't use). It does the actual JWT decode + DB lookup that `Auth` used to do, and is the only handler still touching the cookie directly. Protected only by the shared secret, never by `Auth`/`Require` (that would be circular).
- Local dev without Traefik: see the "Sesión en desarrollo local (sin Traefik)" flow in [README.md](README.md) — you must call `/internal/verify-session` yourself and pass its result as a header on subsequent requests; the cookie alone no longer authenticates anything but that one call.

## Handler Patterns

Handlers follow this pattern:
- Bind/validate request → call app layer → map errors to HTTP status codes using `echo.NewHTTPError`.
- Return `errcustom` sentinel errors (e.g. `errcustom.ErrUnexpectedError`) directly from Echo handlers to let the framework handle them.
- Swagger docs are generated from godoc comments on each handler method using `swag` annotations.

## Documentation Requirements

Every time a handler is created or modified, you **must** update the following — only for what was added or changed:

1. **Swagger comments** on the handler method: keep `@Summary`, `@Description`, `@Param`, `@Success`, and `@Failure` annotations accurate and up to date.
2. **Postman collection** in the [Encargalo.app-api.postman](https://github.com/Encargalo/Encargalo.app-api.postman) repo under `<domain>/<domain>.postman_collection.json`: add or update the corresponding request (method, URL, body, query params, description).

Do **not** regenerate or rewrite documentation for endpoints that were not touched.

> **Postman base URL**: every request URL uses `{{base_url}}`. Do **not** define `base_url` in the collections (`variable`); it lives only in the environments under `environments/` (Local = `http://localhost:3000`).

## Orders & Event Streaming

Order lifecycle events are published to a shared Redis Stream (`ports.RedisStream.PublisherEvent`) from [internal/orders/app/order.go](internal/orders/app/order.go); the stream integration is in [internal/orders/repo/redis_stream.go](internal/orders/repo/redis_stream.go). Current event types: `order_created`, `order_status_changed` (Rejected by the shop or Completed on delivery — carries just `order_id` + `status`, not the full order), `order_auto_rejected`, `order_pending_unassigned`, `payment_proof_pending`. Each consumer reads the same stream with its own consumer group (`redis_stream.group_name` / `consumer_name` in config) so they don't compete for messages — see `cmd/notifications` and `cmd/etl` below.

A credit commission (10% of `totalPrice - deliveryFee`) is sent via HTTP to the `CreditManager` external service when an order reaches `On The Way` status.

### WebSocket — Order Manager

`GET /orders/manager` upgrades to a WebSocket connection (gorilla/websocket). On connect the server sends `{"type":"connected"}`. Supported inbound message types:
- `update_status`: `{"type":"update_status","order_id":"<uuid>","new_status":"<string>"}` — updates the order and broadcasts it to all connected clients.

`OrderManager` role connections are keyed by `shop_id`; all other roles are keyed by `user_id`.

## Trips & Riders

`internal/trips` owns the `Trip` entity and its customer-driven transitions (`RequestTrip`, `CancelTrip`, via REST). Everything the **rider** drives — `accept_trip` and `advance_trip` (`accepted→arriving→in_progress→completed`, plus fare calculation and `RecordTripEarning`) — is implemented in `internal/riders/app` instead, over the WebSocket connection at `GET /orders/rider`, not in `internal/trips/app`.

This is a deliberate split, not an oversight: rider-driven transitions need the same in-memory rider connection/offer state (`appRidersManager`) that `accept_trip` already uses for the "first to accept wins" CAS pattern, so keeping them together avoids threading that state across a domain boundary. `riders/app` reaches `trips` persistence through a narrow `ridersPorts.TripsStore` port (a subset of `tripsPorts.TripsRepo`: `GetTripByID`, `UpdateTripStatus`, `AssignTripRider`, `CompleteTrip`), not through `tripsPorts.TripsApp` — so `trips/app`'s own business rules (e.g. `IsCancellable`) never see these transitions. Keep this split in mind before adding new `Trip` invariants: a rule added only to `trips/app` will not apply to rider-driven transitions.

## Retiros de Riders

`POST /riders/withdrawal` **no paga**: crea una solicitud. El ciclo completo vive en `finance.withdrawal_requests` (genérica por `entity_type`, espejo de `finance.balance_refund_requests`) y es el mismo patrón que los reembolsos de clientes:

```
pending ──claim──> processing ──complete──> completed
   └──────────── reject ────────────┘  (devuelve el saldo retenido)
```

El ledger acompaña con **una sola fila de reserva por solicitud** en `finance.transactions`, vinculada por `reference_type='withdrawal_request'` + `reference_id`, cuyo `movement_type` se muta en cada transición (`withdrawal_hold` → `withdrawal_processing` → `withdrawal_payout` / `withdrawal_restore`). Un índice parcial único sobre `reference_id` garantiza que el saldo no se retenga ni libere dos veces por el mismo retiro.

Tres invariantes que no son obvias y conviene tener presentes antes de tocar este flujo:

- **El `claim` es obligatorio** (no existe `pending → completed`). La transferencia bancaria ocurre fuera del sistema: el operador ve un monto, se va a su app del banco y vuelve minutos después. El reclamo congela `payable_amount`, y el pago registra ese valor **sin recalcular**. Recalcular al pagar haría que los libros digan 72 Bs cuando del banco salieron 800.
- **El monto solicitado es un techo, no una promesa.** Al reclamar, lo pagable se recalcula como `retenido + min(disponible, 0)` (`models.PayableFrom`). Un pedido en efectivo debita fuerte (el rider cobra el total y debe todo lo que no es su pago), así que sin esto la plataforma transferiría 800 Bs a alguien que tiene 728 Bs suyos en el bolsillo. Si el neto queda en cero o menos, la solicitud se anula sola devolviendo la reserva y **no** activa el cooldown.
- **El cooldown de 24h corre desde `completed_at`**, no desde la solicitud, y un rechazo no lo quema. Lo que la regla acota es el costo de una transferencia manual, y ese costo se incurre al pagar. El trabajo anti-spam lo hace el índice parcial único `uq_withdrawal_requests_entity_active`, que impide dos solicitudes activas por entidad a nivel de BD.

El listado del CRM (`GET /riders/staff/finance/withdrawals`) expone `amount_bs` y `payable_bs` **por separado**, más `payable_frozen`. Mostrar un solo número es como se produce un descuadre. Las solicitudes no aparecen en `/riders/staff/finance/payables`, que filtra por saldo > 0: una solicitud retiene el saldo, así que el rider desaparecería de ahí justo cuando hay que pagarle.

El destino de pago sale de `users.accounts.pago_movil` (`GET`/`PUT /users/pago-movil`, habilitado para `Customer` y `Rider`) y se congela en cada solicitud, para que el comprobante corresponda siempre a la cuenta a la que se transfirió.

## ETL / OLAP Worker

`cmd/etl` syncs OLTP → OLAP (`internal/analytics`) two ways at once:
- **Cron** (`robfig/cron`, `ETL_SCHEDULE`): syncs `models.CronETLEntities`, which is now just `{EntityFactTrackingEvent}` — tracking events have no event producer yet, so they still wait for the hourly batch. Every other entity is event-driven and has been fully removed from the cron.
- **Event-driven** (same Redis Stream as `cmd/notifications`, its own consumer group in `startOrderEventConsumer`): `processStreamEvent` dispatches each message's `type` to the `ports.ETLApp.SyncXByID` of the matching entity, which re-reads just that one row from OLTP and upserts it into OLAP immediately — no batch wait. Event types per entity (see `cmd/etl/main.go`):
  - `CubeOrder` ← `order_created`, `order_status_changed`, `order_auto_rejected`
  - `CubeTrip` ← `trip_created`, `trip_status_changed`, `trip_no_driver`
  - `CubeFareQuote` ← `fare_quote_created` (immutable, no "changed" variant)
  - `CubeDeliveryQuote` ← `delivery_quote_created` (immutable, no "changed" variant)
  - `DimShop` ← `shop_updated`, published from `shops/app`
  - `DimRider` ← `rider_updated`, published from `riders/app`
  - `DimUser` ← `user_updated`, published from `users/app` **and** from `riders/app`/`shops/app` when they grant the Rider/Owner role (that mutates `users.user_roles`, which `DimUser.Roles` depends on) — same precedent as `DataEventTrip` being published from both `trips/app` and `riders/app` (see "Trips & Riders" above).

Every event struct lives in the domain that owns the analytical entity (e.g. `DataEventUserUpdated` lives in `users/domain/models` even though `riders/app` and `shops/app` also publish it), not necessarily in the domain that publishes it. Publishers reuse whatever stream port is already wired into that app struct for its own domain's events — no dedicated port per analytical entity.

All sync paths funnel through the same `OLAPSinkRepo.UpsertX` (`INSERT ... ON CONFLICT DO UPDATE`) — the sink never deletes rows. A message type on the stream that doesn't match any of the above is acked and ignored (the stream is shared with other consumers, e.g. `withdrawal_requested`); it is not treated as an error. If `SyncXByID` returns `errcustom.ErrNotFound` (entity deleted or a race between the event and processing), the consumer loop logs a warning and still Acks the message instead of leaving it to be reclaimed and retried forever by `XAutoClaim`.

## Notifications Service

`cmd/notifications` is the consumer of the Redis Stream the API publishes to (`redsumer`, same stream as `internal/orders/repo/redis_stream.go`). It has no Echo server: it reads each `type` field off the stream message and dispatches to `internal/notifications/handler`, which maps the event to a `models.NotificationMessage` in `app` and sends it through `notifier.Fanout` (Slack today; Discord is opt-in via `DISCORD_ENABLED`). Only one plain `net/http` route is exposed, `POST /webhooks/grafana`, which receives Grafana alerts (shared secret + rate limit) and routes them to the prod or dev channel depending on `deployment_environment_name`.

It shares the monorepo's infrastructure: `postgres.NewNotificationsConnection` (OLTP read-only, no migrations), `redis.NewConsumerRedisStreamConnection`, `otelsetup` and `applog` with `reqctx.ServiceNameNotifications`. Its read model lives in `internal/notifications/domain/models` with its own bun structs — it queries the OLTP tables directly instead of going through the other domains' repos.

Deploy: [Dockerfile.notifications](Dockerfile.notifications) (`go build ./cmd/notifications`).

## Testing

- Handler tests use `buildhttptest.SetupControllerCase` from [internal/shared/buildhttptest/](internal/shared/buildhttptest/) to build an Echo context.
- Mocks are generated by `mockery` using [.mockery.yaml](.mockery.yaml) into `internal/mocks/`.
- CI runs on PRs to `main` and `dev` (see [.github/workflows/integration.yaml](.github/workflows/integration.yaml)).

## Environment

Configuration is loaded from env vars at startup via `config.Environments()`. Required groups: `server`, `postgres`, `redis`, `redis_stream`, `jwt`, `environment`, `google`, `cloudinary`, `internal_gateway`. See [internal/shared/config/config.go](internal/shared/config/config.go) for all fields.

Each binary loads only what it needs: `cmd/etl` uses `config.ETLEnvironments()` / `ETLConfig` ([etl_config.go](internal/shared/config/etl_config.go)) and `cmd/notifications` uses `config.NotificationsEnvironments()` / `NotificationsConfig` ([notifications_config.go](internal/shared/config/notifications_config.go)), which adds the `discord`, `slack` and `grafana` groups plus `redis_stream`'s `group_name` / `consumer_name`.
