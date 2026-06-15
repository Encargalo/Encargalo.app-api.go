## ADDED Requirements

### Requirement: Rider puede conectarse al WebSocket de órdenes
El sistema SHALL proveer un endpoint WebSocket `GET /orders/rider` accesible exclusivamente para usuarios con rol `Rider`. Al conectar, el rider entra al pool de disponibles y recibe inmediatamente sus órdenes activas (con `pickup_code`).

#### Scenario: Conexión exitosa
- **WHEN** un usuario con rol `Rider` hace GET `/orders/rider` con sesión válida
- **THEN** se establece la conexión WebSocket y el servidor emite `{"type":"connected"}`

#### Scenario: Rider recibe órdenes activas al conectar
- **WHEN** el rider se conecta y tiene órdenes con `rider_id = <su_id>` en estado "In Preparation", "Ready" o "On The Way"
- **THEN** el servidor emite un mensaje `{"type":"order_update","order":{...}}` por cada orden activa, con `pickup_code` visible y `delivery_code` ausente

#### Scenario: Acceso denegado a otros roles
- **WHEN** un usuario sin rol `Rider` intenta conectarse a `GET /orders/rider`
- **THEN** el servidor retorna HTTP 403

---

### Requirement: Rider recibe nuevas órdenes disponibles
El sistema SHALL emitir un mensaje `new_order` a todos los riders conectados cuando una orden transiciona a "In Preparation". El mensaje no incluye ningún código de confirmación.

#### Scenario: Broadcast de nueva orden disponible
- **WHEN** una orden pasa al estado "In Preparation"
- **THEN** todos los riders en el pool de disponibles reciben `{"type":"new_order","order":{...}}` sin `pickup_code` ni `delivery_code`

#### Scenario: Rider sin conexión no recibe la orden
- **WHEN** una orden pasa a "In Preparation" y no hay riders conectados
- **THEN** no se emite ningún mensaje y el sistema inicia el temporizador de timeout

---

### Requirement: Timeout notifica al servicio de notificaciones
Si ningún rider acepta una orden dentro de los 10 minutos posteriores al broadcast, el sistema SHALL emitir un evento al servicio de notificaciones.

#### Scenario: Timeout sin aceptación
- **WHEN** han transcurrido 10 minutos desde el broadcast de una orden y `rider_id IS NULL`
- **THEN** el sistema emite un evento de notificación indicando que la orden no tiene rider asignado

#### Scenario: Sin timeout si la orden fue aceptada
- **WHEN** un rider acepta la orden antes de que transcurran 10 minutos
- **THEN** el temporizador se cancela y no se emite ningún evento de notificación

---

### Requirement: Rider acepta una orden disponible
El sistema SHALL permitir que un rider envíe `accept_order` para reclamar una orden. Solo el primer rider que acepta es asignado.

#### Scenario: Aceptación exitosa
- **WHEN** el rider envía `{"type":"accept_order","order_id":"<uuid>"}` y la orden no tiene rider asignado
- **THEN** el servidor asigna el rider a la orden y emite `{"type":"order_accepted","order":{...}}` con `pickup_code` visible

#### Scenario: Orden ya tomada por otro rider
- **WHEN** el rider envía `accept_order` pero otro rider ya fue asignado primero
- **THEN** el servidor emite `{"type":"order_taken","order_id":"<uuid>"}` al rider perdedor

#### Scenario: Payload inválido
- **WHEN** el rider envía `accept_order` con `order_id` ausente o malformado
- **THEN** el servidor emite `{"type":"error","message":"payload inválido"}`

---

### Requirement: Rider recibe actualizaciones de sus órdenes activas
El sistema SHALL enviar al rider cualquier actualización de estado de las órdenes que tenga asignadas, con `pickup_code` visible y `delivery_code` ausente.

#### Scenario: Update de orden asignada
- **WHEN** una orden asignada al rider cambia de estado
- **THEN** el rider recibe `{"type":"order_update","order":{...}}` con `pickup_code` y sin `delivery_code`

#### Scenario: Rider no recibe updates de órdenes de otros riders
- **WHEN** una orden asignada a otro rider cambia de estado
- **THEN** el rider no recibe ningún mensaje relacionado con esa orden
