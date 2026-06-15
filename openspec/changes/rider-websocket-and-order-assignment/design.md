## Context

El flujo de órdenes tiene dos transiciones de estado que requieren validación con códigos:
- **In Preparation → On The Way**: el rider presenta `pickup_code` al local.
- **On The Way → Completed**: el customer presenta `delivery_code` al rider.

Actualmente el rol `Rider` existe en el sistema pero no tiene ninguna integración activa: no puede conectarse al WebSocket de órdenes, no hay columna `rider_id` en la tabla de órdenes y no hay mecanismo para notificarle que una orden está disponible.

El WebSocket existente (`/orders/manager`) soporta dos categorías: `shopConnections` (OrderManager/Owner, keyed by `shop_id`) y `customerConnections` (Customer, keyed by `user_id`). La función `stripCodes` filtra los códigos según la categoría pero no conoce el rol Rider.

## Goals / Non-Goals

**Goals:**
- Crear un WebSocket dedicado `/orders/rider` para el rol Rider.
- Mantener un pool en memoria de riders disponibles (conectados).
- Al transicionar una orden a "In Preparation", broadcast de `new_order` (sin códigos) a todos los riders disponibles.
- Asignación atómica de orden a rider mediante UPDATE condicional en DB (`WHERE rider_id IS NULL AND id = ?`).
- Rider ganador recibe `order_accepted` con `pickup_code`; perdedores reciben `order_taken`.
- Un rider puede tener múltiples órdenes activas simultáneamente.
- Rider asignado recibe `order_update` con `pickup_code` en futuros broadcasts de sus órdenes activas.
- Ajustar visibilidad de códigos: Rider → `pickup_code` only; Customer → `delivery_code` only; Shop → ninguno.
- Si ningún rider acepta la orden tras un timeout, emitir un evento al servicio de notificaciones.

**Non-Goals:**
- Geolocalización o matching de riders por proximidad.
- Historial de entregas del rider.
- Rating o reputación de riders.
- Persistir la disponibilidad del rider entre reconexiones.
- Modificar el WebSocket existente `/orders/manager`.

## Decisions

### 1. WebSocket separado para Rider (`/orders/rider`) en lugar de ampliar `/orders/manager`

**Decisión**: nuevo endpoint y nuevo handler `RiderOrderHandler`, con su propio `AppRiderManager`.

**Rationale**: El flujo del rider (pool de disponibles, aceptar orden, race condition, múltiples órdenes activas) es suficientemente distinto al del manager para justificar separación. Mezclarlos añadiría condiciones y complejidad al código existente sin beneficio.

**Alternativa descartada**: Agregar `isRider bool` al handler existente. Rechazado porque obliga a bifurcar la lógica de `Register`, `Unregister`, `BroadcastOrder` con múltiples flags.

---

### 2. Pool de riders en memoria (no en DB)

**Decisión**: `AppRiderManager` mantiene:
- `availableRiders map[uuid.UUID]*WSClient` — riders conectados (keyed by `rider_id`)
- `assignedOrders map[uuid.UUID][]uuid.UUID` — rider → lista de order_ids activos

Ambos protegidos por `sync.RWMutex`.

**Rationale**: La disponibilidad es efímera (rider disponible mientras tiene el WS abierto). Persistirla en DB agrega escrituras innecesarias y complejidad de limpieza.

**Alternativa descartada**: Tabla `riders_available` en Postgres. Requiere limpieza de registros huérfanos. Redis sería razonable para multi-instancia pero está fuera de scope inicial.

---

### 3. Asignación atómica en DB con UPDATE condicional

**Decisión**: `AssignRider(ctx, orderID, riderID)` ejecuta:
```sql
UPDATE orders.orders
SET rider_id = ?, rider_accepted_at = NOW()
WHERE id = ? AND rider_id IS NULL
```
Retorna `ErrOrderAlreadyAssigned` si `rows affected == 0`.

**Rationale**: Soluciona la race condition a nivel de base de datos sin locks de aplicación. Es la técnica estándar para "first write wins" en sistemas concurrentes.

---

### 4. Broadcast a riders en la transición a "In Preparation"

**Decisión**: El broadcast `new_order` ocurre en `orderApp.UpdateOrderStatus` cuando `newStatus == StatusInPreparation`. Si tras un timeout configurable (e.g. 10 min) ningún rider acepta, se emite un evento al servicio de notificaciones.

**Rationale**: En "In Preparation" el local ya aceptó el pedido y está activo. Notificar en `CreateOrder` implicaría que el rider ve pedidos que podrían ser rechazados por el local segundos después.

---

### 5. Dependencia de `AppRiderManager` inyectada en `orderApp`

**Decisión**: `orderApp` recibe `AppRiderManager` como dependencia adicional via DI.

**Rationale**: `orderApp.UpdateOrderStatus` necesita emitir el broadcast al detectar la transición a In Preparation. Retornar un evento y que el handler lo dispatch requeriría cambiar la firma del port y es más invasivo.

---

### 6. `ClientRole` type para `stripCodes`

**Decisión**: Reemplazar `isShop bool` por `type ClientRole string` con valores `ClientRoleShop`, `ClientRoleCustomer`, `ClientRoleRider` en el package `app`.

**Rationale**: `bool` no puede representar tres estados. El tipo nombrado documenta la intención.

---

### 7. Rider con múltiples órdenes activas

**Decisión**: El pool rastrea `assignedOrders map[uuid.UUID][]uuid.UUID` (rider → slice de order_ids). Al reconectar, el rider recibe todas sus órdenes activas. Al aceptar una nueva orden, se agrega al slice existente.

**Rationale**: Un rider eficiente puede llevar varias entregas en paralelo (e.g. mismo edificio). No hay restricción de una orden a la vez.

## Risks / Trade-offs

- **[Race condition a nivel app]** Dos riders envían `accept_order` simultáneamente. → El UPDATE condicional en DB garantiza que solo uno persiste. El perdedor recibe `order_taken` y permanece en el pool disponible.

- **[Estado inconsistente en reconexión]** Si el rider asignado se desconecta y reconecta, necesita recuperar sus órdenes activas. → Al reconectar, `AppRiderManager.Register` consulta `SearchActiveOrdersByRiderID` y envía todas las órdenes activas con `pickup_code`.

- **[Un solo servidor]** El pool en memoria no funciona con múltiples instancias. → Aceptado como limitación inicial. Migración a Redis pub/sub si la escala lo requiere.

- **[Riders inactivos]** Un rider con app en background permanece en el pool. → El ping/pong de 30s del `writePump` detecta la conexión muerta y remueve al rider del pool.

- **[Timeout sin rider]** Si ningún rider acepta en X tiempo, se notifica al servicio de notificaciones. → El timeout se lanza con una goroutine al momento del broadcast; si la orden ya fue aceptada (rider_id != NULL), la goroutine no emite el evento.

## Migration Plan

1. Ejecutar migración goose `00023_order_rider_assignment.sql`:
   - ADD COLUMN `rider_id uuid NULL` en `orders.orders`
   - ADD COLUMN `rider_accepted_at timestamptz NULL` en `orders.orders`
2. Deploy del binario con el nuevo WebSocket y lógica de broadcast.
3. Sin rollback especial: las columnas son nullable, el código existente no las lee.

## Open Questions

- ¿Cuál es el timeout exacto antes de notificar al servicio de notificaciones si ningún rider acepta? Asumido: **10 minutos**. Confirmar con producto.
- ¿Qué canal usa el servicio de notificaciones? (push, Discord, etc.) — El evento se emite como mensaje al servicio existente; el canal es responsabilidad de ese servicio.
