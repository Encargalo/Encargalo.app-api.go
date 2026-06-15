## 1. Base de datos — migración

- [x] 1.1 Crear migración goose `00023_order_rider_assignment.sql`: ADD COLUMN `rider_id uuid NULL` y `rider_accepted_at timestamptz NULL` en `orders.orders`

## 2. Dominio — modelo y ports

- [x] 2.1 Agregar campos `RiderID *uuid.UUID` y `RiderAcceptedAt *time.Time` al modelo `internal/orders/domain/models/order.go`
- [x] 2.2 Agregar método `AssignRider(ctx, orderID, riderID uuid.UUID) error` a la interfaz `OrdersRepo` en `internal/orders/domain/ports/orders.go`
- [x] 2.3 Agregar método `SearchActiveOrdersByRiderID(ctx, riderID uuid.UUID) ([]models.Order, error)` a la interfaz `OrdersRepo`
- [x] 2.4 Agregar interfaz `AppRiderManager` a un nuevo archivo `internal/orders/app/riderManager.go` con métodos `Register`, `Unregister`, `BroadcastNewOrder`, `BroadcastOrderUpdate`

## 3. Repositorio — implementaciones nuevas

- [x] 3.1 Implementar `AssignRider` en `internal/orders/repo/orders.go` con UPDATE condicional `WHERE rider_id IS NULL`; retornar `ErrOrderAlreadyAssigned` si rows affected = 0
- [x] 3.2 Implementar `SearchActiveOrdersByRiderID` en `internal/orders/repo/orders.go` filtrando por `rider_id` y estados activos ("In Preparation", "Ready", "On The Way")

## 4. Errores — sentinel nuevo

- [x] 4.1 Agregar `ErrOrderAlreadyAssigned` en `internal/shared/errcustom/errors.go` y su mapeo HTTP 409 en `errcustom/http.go`

## 5. App — AppRiderManager

- [x] 5.1 Implementar `appRiderManager` en `internal/orders/app/riderManager.go`: pool `availableRiders map[uuid.UUID]*WSClient` y `assignedOrders map[uuid.UUID][]uuid.UUID`, protegidos por `sync.RWMutex`
- [x] 5.2 Implementar `Register`: agrega rider al pool, consulta `SearchActiveOrdersByRiderID` y emite `order_update` con `pickup_code` por cada orden activa
- [x] 5.3 Implementar `Unregister`: remueve rider del pool y de `assignedOrders`
- [x] 5.4 Implementar `BroadcastNewOrder`: emite `{"type":"new_order","order":{...}}` (sin códigos) a todos los riders en `availableRiders`
- [x] 5.5 Implementar `BroadcastOrderUpdate`: emite `{"type":"order_update","order":{...}}` con `pickup_code` a todos los riders en `assignedOrders` cuyo slice contenga el `order_id`
- [x] 5.6 Implementar lógica de aceptación (`AcceptOrder`): llama a `repo.AssignRider`, en éxito agrega a `assignedOrders` y emite `order_accepted`; en fallo emite `order_taken`
- [x] 5.7 Agregar goroutine de timeout de 10 minutos en `BroadcastNewOrder`: si tras 10 min `order.RiderID == nil`, emitir evento al servicio de notificaciones

## 6. App — ajustar stripCodes y BroadcastOrder existente

- [x] 6.1 Definir `type ClientRole string` con constantes `ClientRoleShop`, `ClientRoleCustomer`, `ClientRoleRider` en `internal/orders/app/managerOrder.go`
- [x] 6.2 Actualizar `stripCodes(order, role ClientRole) models.Order`: Shop → oculta ambos; Customer → oculta pickup; Rider → oculta delivery
- [x] 6.3 Actualizar llamadas a `stripCodes` en `BroadcastOrder` y en `Register` de `appManagerOrder`

## 7. App — integrar broadcast en UpdateOrderStatus

- [x] 7.1 Agregar `riderMgr AppRiderManager` como dependencia de `orderApp` en `internal/orders/app/order.go`
- [x] 7.2 En `UpdateOrderStatus`, tras la transición a `StatusInPreparation`, llamar a `riderMgr.BroadcastNewOrder(order)`

## 8. Handler — WebSocket de riders

- [x] 8.1 Crear `internal/orders/handler/riderOrder.go` con interfaz `RiderOrderHandler` y método `HandleWebSocket`
- [x] 8.2 Implementar el loop de lectura WebSocket: manejar mensaje `accept_order` delegando a `appRiderManager.AcceptOrder`; manejar cierre de conexión con `Unregister`
- [x] 8.3 Agregar anotaciones Swagger godoc completas al handler

## 9. Router y DI

- [x] 9.1 Agregar ruta `GET /orders/rider` en `api/router/groups/orders.go` con `Require(models.Rider)`
- [x] 9.2 Registrar `appRiderManager` y `RiderOrderHandler` en `cmd/server/providers/di.go`
- [x] 9.3 Inyectar `AppRiderManager` en `orderApp` en el provide de DI

## 10. Documentación

- [x] 10.1 Regenerar Swagger: `swag init -g cmd/server/main.go --parseDependency --parseInternal`
- [x] 10.2 Agregar request WebSocket `GET /orders/rider` en `postman/orders/orders.postman_collection.json`

## 11. Mocks y tests

- [x] 11.1 Ejecutar `mockery` para regenerar mocks de `OrdersRepo` y `OrdersApp` con los nuevos métodos
- [x] 11.2 Agregar mock de `AppRiderManager` con mockery
- [ ] 11.3 Escribir test unitario para `AssignRider` (éxito y race condition)
- [ ] 11.4 Escribir test unitario para el handler `HandleWebSocket` de riders (conexión, `accept_order` exitoso, `order_taken`)
