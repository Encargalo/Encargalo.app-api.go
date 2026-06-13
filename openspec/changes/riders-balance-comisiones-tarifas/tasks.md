## 1. Prerequisitos y verificación de esquema

- [ ] 1.1 Verificar si existe tabla `riders.riders` o si el perfil del rider está en `users.users`
- [ ] 1.2 Verificar si `orders.orders` tiene columna `distance_km`; si no, agregarla en esta migración
- [ ] 1.3 Determinar el número de secuencia para la nueva migración (siguiente a `00022_`)

## 2. Migración de base de datos

- [ ] 2.1 Crear `internal/shared/adapters/postgres/migrations/000NN_rider_balance.sql` con `-- +goose Up` y `-- +goose Down`
- [ ] 2.2 Agregar columna `balance NUMERIC(10,2) NOT NULL DEFAULT 0` al perfil del rider
- [ ] 2.3 Crear tabla `riders.balance_movements` con todos los campos del diseño
- [ ] 2.4 Ejecutar migración: `goose -dir internal/shared/adapters/postgres/migrations postgres "<DSN>" up`

## 3. Dominio — models y dtos

- [ ] 3.1 Crear `internal/riders/domain/models/rider.go` con structs `Rider` y `BalanceMovement` (bun tags)
- [ ] 3.2 Crear `internal/riders/domain/dtos/balance.go` con `BalanceResponse` (balance + últimos movimientos)
- [ ] 3.3 Crear `internal/riders/domain/dtos/transaction.go` con `TransactionResponse` (lista paginada)
- [ ] 3.4 Crear `internal/riders/domain/dtos/withdrawal.go` con `WithdrawalResponse` (monto retirado + nuevo balance)

## 4. Dominio — ports

- [ ] 4.1 Crear `internal/riders/domain/ports/riders.go` con interfaces `RidersApp` y `RidersRepo`
- [ ] 4.2 Crear `internal/orders/domain/ports/rider_balance.go` con interfaz `RiderBalanceUpdater` (`RecordDelivery`)
- [ ] 4.3 Regenerar mocks: `mockery` (genera mocks para `RidersApp`, `RidersRepo` y `RiderBalanceUpdater`)

## 5. App — lógica de negocio

- [ ] 5.1 Crear `internal/riders/app/riders.go` con constantes de tarifa, comisión y umbrales
- [ ] 5.2 Implementar `GetBalance(ctx, riderID)` — consulta balance + últimos 20 movimientos
- [ ] 5.3 Implementar `GetTransactions(ctx, riderID, page, pageSize)` — historial paginado
- [ ] 5.4 Implementar `RequestWithdrawal(ctx, riderID)` — valida balance ≥ +$15, descuenta y registra
- [ ] 5.5 Implementar `RecordDelivery(ctx, orderID, riderID, paymentMethod, distanceKm, createdAt)` — calcula tarifa, comisión, ajusta balance y crea movimiento (implementa `RiderBalanceUpdater`)
- [ ] 5.6 Crear `internal/riders/app/tariff.go` con función `calculateTariff(createdAt time.Time) float64`

## 6. Repo — persistencia

- [ ] 6.1 Crear `internal/riders/repo/riders.go` implementando `RidersRepo`
- [ ] 6.2 Implementar `GetRiderBalance(ctx, riderID)` — SELECT balance + movimientos recientes
- [ ] 6.3 Implementar `GetTransactions(ctx, riderID, offset, limit)` — SELECT paginado de balance_movements
- [ ] 6.4 Implementar `UpdateBalance(ctx, riderID, delta)` — UPDATE atómico del balance
- [ ] 6.5 Implementar `CreateMovement(ctx, movement)` — INSERT en balance_movements
- [ ] 6.6 Implementar `WithdrawBalance(ctx, riderID, amount)` — UPDATE balance en transacción

## 7. Integración con orders

- [ ] 7.1 Agregar campo `RiderBalanceUpdater ports.RiderBalanceUpdater` (opcional) en struct `orderApp`
- [ ] 7.2 En `ConfirmDelivery` de `internal/orders/app/order.go`, llamar a `RiderBalanceUpdater.RecordDelivery` si no es nil; loguear error sin bloquear la completación
- [ ] 7.3 Actualizar constructor de `orderApp` para aceptar `RiderBalanceUpdater` como parámetro opcional

## 8. Handlers HTTP

- [ ] 8.1 Crear `internal/riders/handler/riders.go` con interfaz `RiderHandler` y struct
- [ ] 8.2 Implementar `GetBalance` — `GET /api/v1/riders/:id/balance` (rol: Rider)
- [ ] 8.3 Implementar `GetTransactions` — `GET /api/v1/riders/:id/transactions` (rol: Rider)
- [ ] 8.4 Implementar `RequestWithdrawal` — `POST /api/v1/riders/:id/withdrawal` (rol: Rider, solo el propio rider)
- [ ] 8.5 Agregar anotaciones Swagger godoc completas en los tres handlers
- [ ] 8.6 Agregar errores nuevos a `internal/shared/errcustom/errors.go`: `ErrInsufficientBalance`, `ErrRiderBalanceRestricted`
- [ ] 8.7 Agregar mapeo HTTP en `internal/shared/errcustom/http.go`: `ErrInsufficientBalance` → 422, `ErrRiderBalanceRestricted` → 403

## 9. Router y DI

- [ ] 9.1 Crear `api/router/groups/riders.go` registrando las tres rutas con middleware `Auth` + `Require(Rider)`
- [ ] 9.2 Registrar `ridersGroup` en `api/router/router.go`
- [ ] 9.3 Agregar providers en `cmd/server/providers/di.go`: `ridersRepo`, `ridersApp`, `riderHandler`, `riderBalanceUpdater`
- [ ] 9.4 Proveer `RiderBalanceUpdater` al constructor de `orderApp` en el contenedor dig

## 10. Tests

- [ ] 10.1 Crear `internal/test/riders/get_balance_test.go`
- [ ] 10.2 Crear `internal/test/riders/get_transactions_test.go`
- [ ] 10.3 Crear `internal/test/riders/request_withdrawal_test.go`
- [ ] 10.4 Crear `internal/test/orders/confirm_delivery_balance_test.go` (verifica que el balance se ajusta al completar)
- [ ] 10.5 Ejecutar `go test ./...` — debe estar verde

## 11. Validación y documentación

- [ ] 11.1 Ejecutar `go build -v ./...` — sin errores
- [ ] 11.2 Regenerar Swagger: `swag init -g cmd/server/main.go --parseDependency --parseInternal`
- [ ] 11.3 Actualizar `postman/riders/riders.postman_collection.json` con los tres endpoints nuevos
- [ ] 11.4 Verificar que los docs Swagger reflejan correctamente los tres endpoints
