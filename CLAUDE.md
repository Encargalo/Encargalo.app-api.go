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

# Build
go build -v ./...

# Run all tests
go test ./...

# Run a single test
go test ./internal/path/to/package -run TestName

# Generate mocks (uses .mockery.yaml config)
mockery

# Generate Swagger docs
swag init -g cmd/server/main.go --parseDependency --parseInternal

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

Domains: `auth`, `customers`, `orders`, `products`, `shops`, `employee`, `health`, `analytics` (ETL → OLAP).

Shared infrastructure lives in `internal/shared/`:
- `config/` — env-based config loaded via `gosuite`, read with `config.Get()`
- `adapters/postgres/` — bun ORM + goose migrations (OLTP en `migrations/`, OLAP en `migrations_olap/`)
- `adapters/redis/` — Redis cache client + stream producer
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

> **Postman base URL**: always use `http://localhost:3000` as the default server URL in the `variable.base_url` field of every Postman collection.

## Orders & Event Streaming

When an order is created or its status transitions to `On The Way`, events are published to a Redis Stream (`ports.RedisStream.PublisherEvent`). The stream integration is in [internal/orders/repo/redis_stream.go](internal/orders/repo/redis_stream.go).

A credit commission (10% of `totalPrice - deliveryFee`) is sent via HTTP to the `CreditManager` external service when an order reaches `On The Way` status.

### WebSocket — Order Manager

`GET /orders/manager` upgrades to a WebSocket connection (gorilla/websocket). On connect the server sends `{"type":"connected"}`. Supported inbound message types:
- `update_status`: `{"type":"update_status","order_id":"<uuid>","new_status":"<string>"}` — updates the order and broadcasts it to all connected clients.

`OrderManager` role connections are keyed by `shop_id`; all other roles are keyed by `user_id`.

## Trips & Riders

`internal/trips` owns the `Trip` entity and its customer-driven transitions (`RequestTrip`, `CancelTrip`, via REST). Everything the **rider** drives — `accept_trip` and `advance_trip` (`accepted→arriving→in_progress→completed`, plus fare calculation and `RecordTripEarning`) — is implemented in `internal/riders/app` instead, over the WebSocket connection at `GET /orders/rider`, not in `internal/trips/app`.

This is a deliberate split, not an oversight: rider-driven transitions need the same in-memory rider connection/offer state (`appRidersManager`) that `accept_trip` already uses for the "first to accept wins" CAS pattern, so keeping them together avoids threading that state across a domain boundary. `riders/app` reaches `trips` persistence through a narrow `ridersPorts.TripsStore` port (a subset of `tripsPorts.TripsRepo`: `GetTripByID`, `UpdateTripStatus`, `AssignTripRider`, `CompleteTrip`), not through `tripsPorts.TripsApp` — so `trips/app`'s own business rules (e.g. `IsCancellable`) never see these transitions. Keep this split in mind before adding new `Trip` invariants: a rule added only to `trips/app` will not apply to rider-driven transitions.

## Testing

- Handler tests use `buildhttptest.SetupControllerCase` from [internal/shared/buildhttptest/](internal/shared/buildhttptest/) to build an Echo context.
- Mocks are generated by `mockery` using [.mockery.yaml](.mockery.yaml) into `internal/mocks/`.
- CI runs on PRs to `main` and `dev` (see [.github/workflows/integration.yaml](.github/workflows/integration.yaml)).

## Environment

Configuration is loaded from env vars at startup via `config.Environments()`. Required groups: `server`, `postgres`, `redis`, `redis_stream`, `jwt`, `environment`, `google`, `cloudinary`, `internal_gateway`. See [internal/shared/config/config.go](internal/shared/config/config.go) for all fields.
