## Why

Los riders son actores independientes del flujo de entrega pero actualmente no tienen ningún punto de integración con la plataforma: no pueden recibir órdenes disponibles, no pueden ver su código de recogida (`pickup_code`) ni completar entregas. Esto bloquea el ciclo completo de una orden.

## What Changes

- **Columnas nuevas** en `orders.orders`: `rider_id` (uuid nullable) y `rider_accepted_at` (timestamp nullable).
- **Nuevo WebSocket** `GET /orders/rider` exclusivo para el rol `Rider`:
  - Al conectar, el rider entra al pool de disponibles y recibe las órdenes activas que ya tiene asignadas (con `pickup_code`).
  - Recibe broadcast de órdenes disponibles (sin códigos) para poder aceptarlas.
  - Mensaje `accept_order` → asignación atómica: solo un rider gana; si ya fue tomada recibe `order_taken`.
- **Broadcast a riders disponibles** desde `UpdateOrderStatus`: cuando una orden transiciona a **"In Preparation"**, se emite `new_order` a todos los riders conectados en el pool.
- **Visibilidad de códigos ajustada**:
  - Rider asignado → ve `pickup_code`, no `delivery_code`.
  - Customer → ve `delivery_code`, no `pickup_code`.
  - Shop → no ve ningún código (sin cambio).

## Capabilities

### New Capabilities

- `rider-websocket`: WebSocket dedicado al rol Rider. Al conectar entra al pool de disponibles; recibe órdenes nuevas cuando pasan a "In Preparation", puede aceptar una (asignación atómica), y recibe updates de sus órdenes activas con `pickup_code`.
- `rider-order-assignment`: Asignación atómica de una orden a un rider mediante UPDATE condicional en DB (`WHERE rider_id IS NULL`). El primer rider que acepta gana; el resto recibe `order_taken`.

### Modified Capabilities

- `order-lifecycle`: La transición a "In Preparation" ahora dispara un broadcast a todos los riders disponibles en el WebSocket. Se agrega `rider_id` al modelo Order.
- `order-confirmation-codes`: La visibilidad de `pickup_code` y `delivery_code` se segmenta por rol (Rider ve pickup, Customer ve delivery, Shop no ve ninguno).

## Impact

- **DB**: migración nueva — columnas `rider_id` y `rider_accepted_at` en `orders.orders`.
- **Dominio orders**: modelo `Order`, ports `OrdersApp`/`OrdersRepo`, app `orderApp` (lógica de broadcast en transición a In Preparation) y nuevo `appRiderManager` (pool de riders + WebSocket).
- **Handler**: nuevo handler `RiderOrderHandler` con WebSocket, registrado en `api/router/groups/orders.go`.
- **DI**: nuevo provide en `cmd/server/providers/di.go`.
- **Router**: nueva ruta `GET /orders/rider` con `Require(models.Rider)`.
- **Swagger + Postman**: anotaciones y request en `postman/orders/orders.postman_collection.json`.
