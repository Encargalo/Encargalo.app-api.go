## 1. Migración de base de datos

- [x] 1.1 Crear migración goose `00024_finance_schema.sql` que crea el schema `finance` con las tablas `balances` y `transactions`
- [x] 1.2 Agregar índices en `finance.transactions(entity_id, entity_type, created_at DESC)` para queries de historial

## 2. Modelos y ports del dominio finance

- [x] 2.1 Crear `internal/riders/domain/models/balance.go` con structs `Balance`, `Transaction` y constantes de `MovementType` y `Zone`
- [x] 2.2 Crear `internal/riders/domain/ports/finance.go` con interfaces `FinanceRepo` (métodos: `GetBalance`, `UpsertBalance`, `CreateTransaction`, `ListTransactions`) y `FinancePort` (método: `RecordRideEarning`)
- [x] 2.3 Agregar `ErrRiderInRiskZone` e `ErrInsufficientBalance` en `internal/shared/errcustom/errors.go` y mapearlos a HTTP en `errcustom/http.go`

## 3. Repositorio finance

- [x] 3.1 Crear `internal/riders/repo/finance.go` implementando `FinanceRepo` con bun ORM sobre `finance.*`
- [x] 3.2 Implementar `GetBalance` con lazy return 0 si no existe fila
- [x] 3.3 Implementar `RecordMovement` con `SELECT FOR UPDATE` + upsert balance + insert transacción en una sola TX
- [x] 3.4 Implementar `ListTransactions` con paginación por `page` y `limit` (máx 50), ordenado por `created_at DESC`

## 4. App layer de riders — lógica de balance

- [x] 4.1 Crear `internal/riders/app/finance.go` implementando `FinanceApp.RecordRideEarning`: delta = ±delivery_fee según método de pago
- [x] 4.2 Nueva fórmula de delivery fee en shops: `distance_km × $0.40/$0.60` reemplaza columnas viejas del shop
- [x] 4.3 Implementar lógica de zona: `getZone(balance float64) Zone`
- [x] 4.4 Implementar `GetBalance`: retorna balance actual, zona y últimos 10 movimientos
- [x] 4.5 Implementar `RequestWithdrawal`: valida balance ≥ +$15, registra withdrawal, balance → 0
- [x] 4.6 Implementar `ListTransactions` con validación de `limit` máx 50

## 5. Integración con orders al completar carrera

- [x] 5.1 Agregar port `RideEarningRecorder` en `internal/orders/domain/ports/orders.go`
- [x] 5.2 Inyectar `RideEarningRecorder` en `internal/orders/app/order.go` vía DI
- [x] 5.3 Llamar a `finance.RecordRideEarning(ctx, order)` en `ConfirmDelivery` tras actualizar status a `Completed`

## 6. Validación de zona de riesgo en AcceptOrder

- [x] 6.1 Inyectar `FinanceRepo` en `internal/riders/app/riders.go`
- [x] 6.2 En `AcceptOrder`: consultar balance; si zona es `risk` y pago en efectivo → `ErrRiderInRiskZone`
- [x] 6.3 En `riders/handler`: mapear `ErrRiderInRiskZone` → `{"type":"error","message":"solo puedes aceptar carreras con pago bancario"}`

## 7. Actualización del WebSocket — datos de ganancia

- [x] 7.1 Agregar `RiderEarnings *float64` y `CollectAmount *float64` en `riderBroadcast`
- [x] 7.2 En `BroadcastNewOrder`: incluir `rider_earnings = delivery_fee × 0.80` y `collect_amount = total_price` (solo efectivo)
- [x] 7.3 En `AcceptOrder`: incluir mismos campos en mensaje `order_accepted`

## 8. Handler HTTP — endpoints de riders

- [x] 8.1 Crear `internal/riders/handler/finance.go` con `GetBalance`, `RequestWithdrawal`, `ListTransactions` y anotaciones Swagger
- [x] 8.2 Rutas en `api/router/groups/orders.go`: `GET /riders/balance`, `POST /riders/withdrawal`, `GET /riders/transactions` con `Auth + Require(Rider)`

## 9. Dependency Injection

- [x] 9.1 Registrar `FinanceRepo`, `FinanceApp`, `FinanceHandler` y adapter `RideEarningRecorder` en `cmd/server/providers/di.go`
- [x] 9.2 `NewOrderApp` recibe `RideEarningRecorder` como dependencia

## 10. Mocks y generación

- [x] 10.1 `.mockery.yaml` ya incluía `riders/domain/ports` — cubre `FinanceRepo` y `FinanceApp`
- [x] 10.2 `mockery` ejecutado — mocks generados en `internal/mocks/riders/domain/ports/`

## 11. Documentación

- [x] 11.1 `swag init` regenerado con los 3 endpoints nuevos
- [x] 11.2 3 endpoints agregados en `postman/orders/orders.postman_collection.json`
