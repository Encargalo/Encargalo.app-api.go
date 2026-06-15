## ADDED Requirements

### Requirement: La tabla de órdenes almacena el rider asignado
El sistema SHALL persistir `rider_id` y `rider_accepted_at` en `orders.orders`. Ambas columnas son nullable; una orden sin rider tiene `rider_id = NULL`.

#### Scenario: Orden recién creada sin rider
- **WHEN** se crea una orden
- **THEN** `rider_id` es NULL y `rider_accepted_at` es NULL

#### Scenario: Rider asignado correctamente
- **WHEN** un rider acepta la orden y la asignación es exitosa
- **THEN** `rider_id` contiene el UUID del rider y `rider_accepted_at` contiene el timestamp de aceptación

---

### Requirement: Asignación atómica mediante UPDATE condicional
El sistema SHALL asignar una orden a un rider usando `UPDATE ... WHERE rider_id IS NULL` para garantizar que solo un rider puede ganar en condiciones de concurrencia.

#### Scenario: Primer rider gana la asignación
- **WHEN** dos riders intentan aceptar la misma orden simultáneamente
- **THEN** exactamente uno recibe la asignación (rows affected = 1) y el otro recibe `ErrOrderAlreadyAssigned`

#### Scenario: Intento de asignar orden ya asignada
- **WHEN** `AssignRider` se ejecuta para una orden cuyo `rider_id` ya no es NULL
- **THEN** retorna `ErrOrderAlreadyAssigned` sin modificar el registro

---

### Requirement: Un rider puede tener múltiples órdenes activas
El sistema SHALL permitir que un mismo rider sea asignado a más de una orden simultáneamente. No existe límite de órdenes activas por rider.

#### Scenario: Rider acepta segunda orden
- **WHEN** un rider que ya tiene una orden activa acepta otra orden disponible
- **THEN** ambas órdenes quedan asignadas al mismo rider y el rider las recibe en el WebSocket

#### Scenario: Rider recibe todas sus órdenes activas al reconectar
- **WHEN** un rider con múltiples órdenes activas se desconecta y vuelve a conectar
- **THEN** el servidor emite `order_update` por cada orden activa del rider al reconectar

---

### Requirement: Búsqueda de órdenes activas por rider
El sistema SHALL proveer un método de repositorio `SearchActiveOrdersByRiderID` que retorne las órdenes con `rider_id = <id>` en estado activo ("In Preparation", "Ready", "On The Way").

#### Scenario: Rider con órdenes activas
- **WHEN** se consulta `SearchActiveOrdersByRiderID` con un rider que tiene órdenes activas
- **THEN** retorna la lista de órdenes activas de ese rider

#### Scenario: Rider sin órdenes activas
- **WHEN** se consulta `SearchActiveOrdersByRiderID` con un rider sin órdenes activas
- **THEN** retorna `ErrNotFound`
