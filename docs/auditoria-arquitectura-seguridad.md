# Auditoría de Arquitectura y Seguridad — Encargalo.app-api.go

**Fecha**: 2026-08-15  
**Alcance**: Base de código completa (`internal/`, `api/`, `cmd/`)  
**Referencias**: `/reglas-arquitectura`, `/auditor-arquitectura`, `/auditor-seguridad`, OWASP API Security Top 10  
**Método**: Revisión estática de capas (domain/app/repo/handler), DI, routing, auth/gateway, órdenes/WS, pricing/comisiones, logging y configuración

---

## Progreso de remediación

| ID | Severidad | Estado | Notas |
|----|-----------|--------|-------|
| A-8 | MAJOR | ✅ Corregido | `ComputeShopSaleBreakdown` en `shops/domain/models/finance.go` — usado por `shops/app/finance.go` y `orders/app/managerOrder.go` |
| A-1 | CRITICAL | ⬜ Pendiente | |
| A-2 | CRITICAL | ⬜ Pendiente | |
| A-3 | CRITICAL | ⬜ Pendiente | |
| A-4 | MAJOR | ⬜ Pendiente | |
| A-5 | MAJOR | ⬜ Pendiente | |
| A-6 | MAJOR | ⬜ Pendiente | |
| A-7 | MAJOR | ⬜ Pendiente | |
| A-9 | MAJOR | ⬜ Pendiente | |
| A-10 | MAJOR | ⬜ Pendiente | |
| A-11 | MAJOR | ⬜ Pendiente | |
| A-12 | MAJOR | ⬜ Pendiente | |
| A-13 | MAJOR | ⬜ Pendiente | |
| A-14 | MAJOR | ⬜ Pendiente | |
| S-1 | CRITICAL | ⬜ Pendiente | |
| S-2 | HIGH | ⬜ Pendiente | |
| S-3 | HIGH | ⬜ Pendiente | |
| S-4 | HIGH | ⬜ Pendiente | |
| S-5 | HIGH | ⬜ Pendiente | |
| S-6 | HIGH | ⬜ Pendiente | |

---

## Resumen ejecutivo

El esqueleto hexagonal está en buen estado: handlers dependen de `ports.XxxApp`, apps de `ports.XxxRepo`, DI concentrada en dig, transacciones en repos y uso amplio de `errcustom`.

La deuda bloqueante se concentra en:

1. **`shared/` importando dominios** (`orders`, `bcv`, `auth`)
2. **Middleware Echo dentro de `auth/domain/`**
3. **Reglas comerciales (promo, delivery fee, comisión) en repos/SQL**
4. **Riesgos de seguridad HIGH**: CORS wildcard+credentials, sin body limit, SQL debug siempre on, FSM de órdenes incompleta, JWT/sesión de 1 año

| Área | CRITICAL | HIGH/MAJOR | MEDIUM/MINOR | INFO |
|------|----------|------------|--------------|------|
| Arquitectura | 3 | 11 | 6 | 5 |
| Seguridad | 1 | 5 | 6 | 2 (+2 LOW) |

---
	
# Parte A — Auditoría de Arquitectura

## CRITICAL

### Hallazgo A-1

- **Severidad**: CRITICAL
- **Regla afectada**: §1 — `shared/` NO DEBE importar módulos de dominio
- **Archivo**: `internal/shared/deliveryfee/bcv.go` líneas 7–19
- **Evidencia**:
```go
bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
// ...
func ApplyBsToOrder(ctx context.Context, bcv bcvPorts.BcvApp, order *ordersModels.Order) {
	order.ApplyBsPrice(rate.Rate)
}
```
- **Problema**: Infraestructura compartida acoplada a `orders` y `bcv`.
- **Impacto**: Ciclos de dependencia; `shared` deja de ser genérico.
- **Propuesta**: Mover `ApplyBsToOrder` a `orders/app`. Dejar en `shared/deliveryfee` solo tarifas/distancia sin tipos de dominio.

### Hallazgo A-2

- **Severidad**: CRITICAL
- **Regla afectada**: §1 — `shared/` NO DEBE importar dominios
- **Archivo**: `internal/shared/reqctx/reqctx.go` líneas 6, 65
- **Evidencia**:
```go
authmodels "Encargalo.app-api.go/internal/auth/domain/models"
// ...
session, ok := req.Context().Value(authmodels.SessionKey).(authmodels.ActiveSession)
```
- **Problema**: `shared/reqctx` depende de modelos de `auth`.
- **Impacto**: Cambios de sesión rompen infraestructura compartida.
- **Propuesta**: Extraer `SessionKey` + DTO mínimo de sesión a `shared`, o inyectar IDs ya resueltos sin importar `auth`.

### Hallazgo A-3

- **Severidad**: CRITICAL
- **Regla afectada**: §1 / §2 — `domain/` sin HTTP; capas invertidas
- **Archivo**: `internal/auth/domain/middleware/auth/auth.go` líneas 1–74
- **Evidencia**:
```go
package auth
import (
	"github.com/labstack/echo/v4"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
)
func (a *auth) Auth(next echo.HandlerFunc) echo.HandlerFunc { ... echo.NewHTTPError(...) }
```
- **Problema**: Middleware HTTP (Echo) vive bajo `domain/`.
- **Impacto**: El dominio de auth depende del framework web.
- **Propuesta**: Mover a `internal/auth/middleware/` o `internal/shared/middleware/auth/`. En domain solo `models.Role` / `ActiveSession`.

---

## MAJOR

### Hallazgo A-4

- **Severidad**: MAJOR
- **Regla afectada**: §2 — `repo/` NO DEBE contener lógica de negocio
- **Archivo**: `internal/shops/repo/shops.go` ~L29–38, L627–640
- **Evidencia**:
```go
launchPromoEndsAt := now.Add(launchPromoWindow)
zeroCommissionEndsAt := now.Add(zeroCommissionWindow)
```
- **Problema**: Ventanas de promo / comisión 0% se anclan en el repo.
- **Impacto**: Reglas comerciales no testeables vía app; riesgo de divergencia.
- **Propuesta**: Calcular timestamps en `shops/app` (o método de dominio) y pasarlos al repo.

### Hallazgo A-5

- **Severidad**: MAJOR
- **Regla afectada**: §2 — lógica de negocio en `repo/`
- **Archivo**: `internal/shops/repo/stats.go` línea 65
- **Evidencia**:
```go
ColumnExpr("COUNT(*) AS cnt, COALESCE(SUM((total_price - delivery_fee) * 0.9), 0) AS lost")
```
- **Problema**: Share 90% / comisión 10% hardcodeada en SQL.
- **Impacto**: Diverge de `ShopShare` / `shopCommissionRate` si cambia la tarifa.
- **Propuesta**: Usar constante de dominio o calcular “lost” en app a partir de filas crudas.

### Hallazgo A-6

- **Severidad**: MAJOR
- **Regla afectada**: §2 — lógica de negocio en `repo/`
- **Archivo**: `internal/products/repo/products.go` líneas 620–631 (y fees/ETA ~710+)
- **Evidencia**:
```go
if hasLaunchPromo {
	effectiveRef = money.RoundBs(ref * 0.5)
}
```
- **Problema**: Repo aplica promo 50%, tarifa delivery y prep time.
- **Impacto**: Pricing fuera de app; duplicado vs shops/orders.
- **Propuesta**: Repo retorna datos crudos; `products/app` arma `TrendingResponse`.

### Hallazgo A-7

- **Severidad**: MAJOR
- **Regla afectada**: §2 / §9 — app sin infra directa; config/ports
- **Archivo**: `internal/orders/app/order.go` (vía `deliveryfee.Calculate` → Google Maps)
- **Problema**: App llama Google Maps a través de `shared/deliveryfee` sin puerto inyectable.
- **Impacto**: Tests difíciles sin red; viola “solo ports”.
- **Propuesta**: Puerto `DeliveryFeeCalculator` / `DistanceProvider` en dig.

### Hallazgo A-8 ✅ Corregido (2026-08-15)

- **Severidad**: MAJOR
- **Regla afectada**: §9 / §2 — no hardcodear; no duplicar reglas
- **Archivo**: `internal/orders/app/managerOrder.go` L25, L177–186 vs `internal/shops/domain/models/finance.go`
- **Evidencia** (antes):
```go
const shopCommissionRate = 0.10
// vs shops: ShopShare = 0.90
```
- **Problema**: Comisión duplicada en dos sitios.
- **Impacto**: Broadcast WS puede no coincidir con liquidación real.
- **Remediación aplicada**: `ComputeShopSaleBreakdown` en `shops/domain/models/finance.go`; usado por `shops/app/finance.go` y `orders/app/managerOrder.go`. Tests en `internal/test/shops/models/finance_test.go`.

### Hallazgo A-9

- **Severidad**: MAJOR
- **Regla afectada**: §2 — domain sin dependencias de infraestructura
- **Archivo**: `internal/products/domain/models/data_produce_order.go` líneas 7–24
- **Evidencia**:
```go
"github.com/redis/go-redis/v9"
func (d *DataProduceOrder) BuildModel(message redis.XMessage) error { ... }
```
- **Problema**: Modelo de dominio depende de `go-redis`.
- **Impacto**: Domain acoplado a vendor de streams.
- **Propuesta**: Mapping en repo/app con `map[string]any` o DTO propio.

### Hallazgo A-10

- **Severidad**: MAJOR
- **Regla afectada**: §2 — pricing en mapping de modelo
- **Archivo**: `internal/shops/domain/models/shops/shops.go` (~delivery fee / ETA al mapear)
- **Problema**: Entity de shop calcula delivery fee y ETA al mapear a DTO.
- **Impacto**: Reglas escondidas en domain↔DTO; duplicadas en products repo.
- **Propuesta**: Mapping de fee/ETA en `shops/app` o servicio de pricing vía puerto.

### Hallazgo A-11

- **Severidad**: MAJOR
- **Regla afectada**: §5 / §2 — handler: bind→app→map; sin orquestación de negocio
- **Archivo**: `internal/orders/handler/order.go` L163–169; `managerOrder.go` L94–102
- **Evidencia**:
```go
employee, err := o.employee.SearchEmployeeByID(ctx, session.UserID)
order, err := o.svc.ConfirmPickup(ctx, orderID, req.Code, employee.ShopID)
```
- **Problema**: Resolución empleado→shop y decisión de rol de conexión en handler.
- **Impacto**: Autorización/orquestación fuera de app; más difícil de testear.
- **Propuesta**: `ConfirmPickup(ctx, orderID, code, userID)` y registro WS en app; handler solo pasa sesión.

### Hallazgo A-12

- **Severidad**: MAJOR
- **Regla afectada**: §9 — logging vía `applog`; charmbracelet deprecado
- **Archivo**: `internal/shared/adapters/redis/producer.go`
- **Evidencia**: `github.com/charmbracelet/log` + `log.Fatalf`
- **Problema**: Único uso vivo de charmbracelet/log.
- **Impacto**: Formato inconsistente; dependencia deprecada.
- **Propuesta**: Reemplazar por `slog`/`applog`; limpiar `go.mod` si aplica.

### Hallazgo A-13

- **Severidad**: MAJOR
- **Regla afectada**: §9 — no loguear PII
- **Archivo**: `internal/users/handler/users.go` L70; `internal/shops/handler/staffShop.go` L58
- **Evidencia**: logs con `"phone", customer.Phone` / `phone`
- **Problema**: Teléfonos en claro en logs.
- **Impacto**: Riesgo de privacidad / compliance.
- **Propuesta**: Loguear hash truncado o solo `user_id`.

### Hallazgo A-14

- **Severidad**: MAJOR
- **Regla afectada**: §2 — repo no orquesta pricing/geo
- **Archivo**: `internal/shops/repo/shops.go` (imports `googlemaps` + `deliveryfee`)
- **Problema**: Persistencia mezclada con pricing y llamadas externas.
- **Impacto**: Repo difícil de mockear; reglas en dos capas.
- **Propuesta**: Repo retorna shops + coords; app (o puerto `DistanceMatrix`) calcula fees.

---

## MINOR

### Hallazgo A-15

- **Severidad**: MINOR
- **Regla afectada**: §5 — Swagger en handlers
- **Archivo**: `internal/health/handler/health.go`
- **Problema**: Sin anotaciones Swagger (aceptable como health, inconsistente).
- **Propuesta**: Anotaciones mínimas o excepción documentada.

### Hallazgo A-16

- **Severidad**: MINOR
- **Regla afectada**: §9 — slog vía `applog`
- **Archivos**: Amplio uso de `log/slog` directo (orders, riders, shops, products, bcv, …) mientras auth/users usan `applog`.
- **Problema**: Doble estándar; se pierde request meta de `applog`.
- **Propuesta**: Migrar progresivamente en código tocado.

### Hallazgo A-17

- **Severidad**: MINOR
- **Regla afectada**: §5 — handler sin side-effects de negocio
- **Archivo**: `internal/orders/handler/order.go` (~branch `StatusPaymentReview` + broadcast)
- **Problema**: Decisión de broadcast parcial en handler.
- **Propuesta**: App publica el evento apropiado o retorna flag tipado.

### Hallazgo A-18

- **Severidad**: MINOR
- **Regla afectada**: §9 — URLs hardcodeadas
- **Archivos**: `bcv/repo/scraper.go`, `calendar_scraper.go`, `shared/adapters/googlemaps/*`
- **Problema**: Endpoints fijos (keys sí vienen de config).
- **Propuesta**: Opcional `config.*.BaseURL` con defaults.

### Hallazgo A-19

- **Severidad**: MINOR
- **Regla afectada**: §2 — DTOs sin lógica de negocio
- **Archivos**: Varios `Validate()` OK; `users/domain/dto/address.go` `BuildExtra` / `UnmarshalJSON`
- **Problema**: Construcción polimórfica de `Extra` roza dominio.
- **Propuesta**: Mantener `Validate()`; mover construcción a app/modelo.

### Hallazgo A-20

- **Severidad**: MINOR
- **Regla afectada**: §9 — tarifas duplicadas
- **Archivos**: `shared/deliveryfee` (`RateDay`/`RateNight`) vs `riders/domain/models/balance.go`
- **Problema**: Mismas constantes en dos sitios.
- **Propuesta**: Una fuente de verdad.

---

## INFO (arquitectura)

### Hallazgo A-21

- **Severidad**: INFO
- **Observación**: Wiring de capas limpio: handlers → `ports.XxxApp`, apps → `ports.XxxRepo`. Sin imports concretos `handler→app` ni `app→repo`. DI en `cmd/server/providers/di.go` y `cmd/etl/providers/di.go`.

### Hallazgo A-22

- **Severidad**: INFO
- **Observación**: `employee` sin handler (OK). `analytics` sin handler de negocio (OK). Sesión vía `models.SessionKey`. `RunInTx` en repos (OK).

### Hallazgo A-23

- **Severidad**: INFO
- **Archivo**: `riders/domain` importa models de `orders` (documentado como deliberado)
- **Propuesta**: Aceptable en modular monolith; a medio plazo DTO anti-corrupción.

### Hallazgo A-24

- **Severidad**: INFO
- **Observación**: No se encontró concatenación de input de usuario en SQL. Bun con `?` predominante.

### Hallazgo A-25

- **Severidad**: INFO
- **Observación**: Apps no devuelven `echo.NewHTTPError`. Uso amplio de `errcustom` + mapeo HTTP.

---

## Antipatrones Go / convenciones adicionales

| # | Severidad | Antipatrón | Ubicación | Nota |
|---|-----------|------------|-----------|------|
| G-1 | MINOR | Typos en API pública (`NewAuthMidlleware`, `NewProducsHandler`, `ErrUnautorized`) | auth middleware, products handler, errcustom | Deuda de naming; renombrar con alias deprecado si se exporta |
| G-2 | MINOR | Constructores / nombres inconsistentes en español-inglés mezclado | varios handlers | Cosmético |
| G-3 | INFO | Panic implícito en domain (`uuid.MustParse` en `BuildModel`) | `data_produce_order.go` | Preferir error explícito |
| G-4 | INFO | Type assertions sin check en Redis message values | mismo archivo | Puede panic con payload malformado |

---

## Resumen por módulo (arquitectura)

| Módulo | Estado | Notas |
|--------|--------|-------|
| auth | Debilidad estructural | Middleware Echo en `domain/` (A-3) |
| users | OK con matices | PII en logs (A-13); auto-login en handler |
| shops | Problemas de capa | Promo windows + fees en repo/domain (A-4, A-10, A-14); stats `*0.9` (A-5) |
| products | Problemas de capa | Promo/delivery en repo (A-6); redis en domain (A-9) |
| orders | Orquestación / ports | deliveryfee sin puerto (A-7); commission duplicada (A-8); employee en handler (A-11) |
| employee | Limpio | Solo app→repo ports |
| payment_methods | Limpio | Passthrough correcto |
| bcv | OK | URLs hardcodeadas (A-18) |
| health | Mínimo | Sin Swagger (A-15) |
| analytics | Limpio | Ports OLTP/OLAP correctos |
| riders | OK | Cross-domain orders (A-23); rates duplicados (A-20) |
| maps | Limpio | Validación en app |
| shared | **Bloqueante** | Imports a dominios (A-1, A-2); charmbracelet (A-12) |

## Veredicto de Arquitectura

**Estado**: Bloqueado por refactor

**Resumen**:
- CRITICAL: 3
- MAJOR: 11
- MINOR: 6
- INFO: 5 (+ antipatrones Go menores)

Aplicar refactor sobre CRITICAL (A-1–A-3) y MAJOR de pricing/comisión en repos (A-4–A-8, A-10, A-14) antes de features que toquen fees, promos o comisiones.

---

# Parte B — Auditoría de Seguridad

## CRITICAL

### Vulnerabilidad S-1

- **Severidad**: CRITICAL
- **Categoría OWASP**: API2 Broken Authentication / API5
- **Archivo**: `internal/auth/domain/middleware/auth/auth.go` L31–49; `internal/shared/middleware/gatewaysecret.go`
- **Evidencia**: Tras el secreto de gateway, `Auth` unmarshalea JSON unsigned de `X-Encargalo-Session` (roles, `user_id`, `shop_id`) sin revalidar JWT/DB/`ExpiresAt`.
- **Vulnerabilidad**: Confianza ciega en header de sesión forjable si se filtra el gateway secret.
- **Vector de ataque**: Quien conozca `X-Internal-Gateway-Secret` forja sesión Staff/Owner de cualquier tienda.
- **Impacto**: Suplantación total de identidad y privilegios.
- **Remediación**: Firmar el header (HMAC/JWT corto), o revalidar `session.id` en DB; rotar secreto; no tratar el JSON como única source of truth.

> **Nota de diseño**: El patrón Traefik ForwardAuth + gateway secret es deliberado. El hallazgo permanece CRITICAL por el blast radius si el secreto se filtra (misconfig, logs, acceso a red privada).

---

## HIGH

### Vulnerabilidad S-2

- **Severidad**: HIGH
- **Categoría OWASP**: API6
- **Archivo**: `internal/orders/app/order.go` L99–106, L654–696
- **Evidencia**: `allowedTransitionStatuses` es un set de destinos; no hay FSM from→to. Shop puede saltar a `On The Way` / `Rejected` sin validar estado actual.
- **Vulnerabilidad**: Máquina de estados ausente.
- **Vector de ataque**: WS `update_status` con destino arbitrario; bypass de pickup code / payment review.
- **Impacto**: Flujos de pago/pickup saltados; reembolsos indebidos; riders notificados prematuramente.
- **Remediación**: FSM explícita; `On The Way` solo vía `ConfirmPickup`; bloquear cambios desde `Payment Review` salvo Staff.

### Vulnerabilidad S-3

- **Severidad**: HIGH
- **Categoría OWASP**: API8
- **Archivo**: `api/router/router.go` L68–74
- **Evidencia**:
```go
AllowOrigins: []string{"*"},
AllowCredentials: true,
UnsafeWildcardOriginWithAllowCredentials: true,
```
- **Vulnerabilidad**: CORS permite cualquier origen con credenciales.
- **Vector de ataque**: Sitio malicioso + cookie (especialmente non-prod con SameSite=None).
- **Impacto**: CSRF / lectura de respuestas autenticadas.
- **Remediación**: Allowlist de orígenes; quitar `UnsafeWildcardOriginWithAllowCredentials`.

### Vulnerabilidad S-4

- **Severidad**: HIGH
- **Categoría OWASP**: API4
- **Archivo**: `api/router/router.go` (sin `BodyLimit`); uploads base64 sin tope previo
- **Vulnerabilidad**: Requests/JSON/base64 arbitrariamente grandes.
- **Vector de ataque**: POST con imagen base64 masiva.
- **Impacto**: DoS de memoria/CPU; coste Cloudinary.
- **Remediación**: `BodyLimit` global (2–5MB); límite de bytes decodificados; rate limit en uploads.

### Vulnerabilidad S-5

- **Severidad**: HIGH
- **Categoría OWASP**: API8 / Logging
- **Archivo**: `internal/shared/adapters/postgres/postgres.go` L26–40
- **Evidencia**: `sslmode=disable` + `bundebug.WithVerbose(true)` siempre activo.
- **Vulnerabilidad**: SQL debug en todos los entornos; Postgres sin TLS en DSN.
- **Vector de ataque**: Logs con queries/params (PII); MITM si la red no es privada.
- **Impacto**: Filtración de datos; compromiso de credenciales en tránsito.
- **Remediación**: `bundebug` solo non-prod; `sslmode=require` (o verify-full) en producción.

### Vulnerabilidad S-6

- **Severidad**: HIGH
- **Categoría OWASP**: API2
- **Archivo**: `internal/shared/jwt/jwt.go` L39; sesión/cookie ~1 año
- **Vulnerabilidad**: TTL excesivo sin rotación aparente.
- **Vector de ataque**: Robo de cookie → acceso prolongado.
- **Impacto**: Ventana de abuso de meses.
- **Remediación**: TTL corto + refresh; re-auth para acciones sensibles.

---

## MEDIUM

### Vulnerabilidad S-7

- **Severidad**: MEDIUM
- **Categoría OWASP**: API9
- **Archivo**: `api/router/router.go` L77 — `/docs/*` sin auth
- **Vulnerabilidad**: Swagger público expone inventario de API.
- **Remediación**: Deshabilitar en prod o proteger.

### Vulnerabilidad S-8

- **Severidad**: MEDIUM
- **Categoría OWASP**: API3 / Error Handling
- **Archivo**: handlers WS (`managerOrder`, `staffReviews`, finance)
- **Vulnerabilidad**: `err.Error()` enviado al cliente WS.
- **Remediación**: Mensajes seguros (como `safeErrorMessage` en riders).

### Vulnerabilidad S-9

- **Severidad**: MEDIUM
- **Categoría OWASP**: API4
- **Archivo**: `shared/middleware/ratelimit.go` (fail-open); `POST /users` sin rate limit
- **Vulnerabilidad**: Sin límite si Redis falla; registro ilimitado.
- **Remediación**: Fail-closed o límite local; rate limit en registro.

### Vulnerabilidad S-10

- **Severidad**: MEDIUM
- **Categoría OWASP**: API4 / Business logic
- **Archivo**: códigos de 4 dígitos + rate limit solo por IP
- **Vulnerabilidad**: Espacio pequeño; fuerza bruta con rotación de IP.
- **Remediación**: 6+ dígitos o alfanumérico; rate limit por `order_id`+actor.

### Vulnerabilidad S-11

- **Severidad**: MEDIUM
- **Categoría OWASP**: API8 / WebSocket
- **Archivo**: `internal/shared/ws/upgrader.go`
- **Vulnerabilidad**: CheckOrigin permisivo (Origin vacío → true en prod).
- **Remediación**: Rechazar Origin vacío salvo allowlist de clientes nativos.

### Vulnerabilidad S-12

- **Severidad**: MEDIUM
- **Categoría OWASP**: API6
- **Archivo**: `CreateOrder` — cliente elige `id` del pedido
- **Vulnerabilidad**: IDs controlados por el cliente.
- **Remediación**: Generar `order.ID` solo en servidor.

### Vulnerabilidad S-13

- **Severidad**: MEDIUM
- **Categoría OWASP**: API3
- **Archivo**: DTOs de shops con `home_phone` en listados customer
- **Vulnerabilidad**: PII de negocio en catálogo amplio.
- **Remediación**: Omitir en listados; exponer solo si es necesario en detalle.

---

## LOW / INFO (seguridad)

### Vulnerabilidad S-14 (LOW)

- **Categoría OWASP**: API3 / Logging — PII (`phone`) en logs de staff shop search.

### Vulnerabilidad S-15 (LOW)

- **Categoría OWASP**: API4 — `GetOrderHistory` sin paginación.

### Vulnerabilidad S-16 (INFO)

- **Categoría OWASP**: API5 — `payment_methods` solo con `Auth`, sin `Require(role)` (dato probablemente público).

### Vulnerabilidad S-17 (INFO)

- **Categoría OWASP**: API10 — CreditManager externo ya no aparece en código; comisión interna `RecordSaleCommission`. URLs externas fijas/config (sin SSRF por input de usuario observado).

---

## Controles que están bien

| Área | Estado |
|------|--------|
| Gateway secret en `Auth` + `verify-session` (constant-time) | OK |
| Cookie `HttpOnly` + `Secure`; SameSite Strict en prod | OK |
| JWT HMAC + validación de alg | OK |
| BOLA orders: ownership shop/customer/rider en update/confirm | OK (salvo FSM) |
| Productos business: `session.ShopID` + check `item.ShopID` | OK (si sesión no forjada) |
| Address delete / refund get con ownership | OK |
| Precios/comisión server-side (quote + catálogo) | OK |
| SQL bun parametrizado | OK |
| UUID parse en la mayoría de path params | OK |
| `errcustom.HTTPErrorHandler` no filtra causas internas (HTTP) | OK |
| Rate limit en sign-in, maps, confirm codes, staff candidates | OK |
| Roles en rutas Owner/Staff/Rider/OrderManager | En general OK |

## Veredicto de Seguridad

**Estado**: Requiere remediación antes de deploy

**Resumen**:
- CRITICAL: 1
- HIGH: 5
- MEDIUM: 6
- LOW: 2
- INFO: 2

Corregir CRITICAL y HIGH antes de exponer en producción: (1) no confiar en JSON unsigned de sesión sin revalidación/firma, (2) FSM de órdenes, (3) CORS allowlist, (4) body/upload limits, (5) apagar SQL debug + TLS Postgres, (6) acortar vida de JWT/sesión.

---

# Priorización sugerida

### P0 — Bloqueantes inmediatos
1. CORS allowlist (S-3)
2. Apagar `bundebug` en prod + TLS DSN (S-5)
3. `BodyLimit` + límites de upload (S-4)
4. FSM de órdenes (S-2)

### P1 — Arquitectura / deuda estructural
5. Mover middleware fuera de `auth/domain` (A-3)
6. Eliminar imports `shared→domain` (A-1, A-2)
7. Sacar pricing/promo/comisión de repos (A-4, A-5, A-6, A-14)
8. Unificar tasa de comisión (A-8)

### P2 — Hardening
9. Firmar/revalidar header de sesión (S-1)
10. Acortar TTL JWT/sesión (S-6)
11. Rate limit registro + fail-closed (S-9)
12. PII fuera de logs (A-13 / S-14)
13. Puerto inyectable para delivery fee (A-7)

---

*Generado con `/auditor-arquitectura` + `/auditor-seguridad` sobre el árbol completo del repositorio.*
