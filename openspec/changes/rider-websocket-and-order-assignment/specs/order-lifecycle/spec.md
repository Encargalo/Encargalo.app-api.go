## MODIFIED Requirements

### Requirement: Transición a "In Preparation" notifica a riders disponibles
Al transicionar una orden a estado "In Preparation", el sistema SHALL emitir un broadcast `new_order` a todos los riders disponibles conectados al WebSocket de riders, además de generar el `pickup_code` como ya ocurría.

#### Scenario: Broadcast a riders al entrar en preparación
- **WHEN** `UpdateOrderStatus` transiciona una orden a "In Preparation"
- **THEN** se genera el `pickup_code`, se persiste, y se emite `new_order` a todos los riders en el pool de disponibles

#### Scenario: Sin riders disponibles no hay error
- **WHEN** `UpdateOrderStatus` transiciona a "In Preparation" y no hay riders conectados
- **THEN** la transición de estado y la generación del `pickup_code` se completan con éxito; el broadcast simplemente no tiene destinatarios

---

### Requirement: El modelo Order incluye rider_id y rider_accepted_at
El modelo de dominio `Order` SHALL incluir los campos `RiderID` (uuid nullable) y `RiderAcceptedAt` (timestamp nullable).

#### Scenario: Orden sin rider asignado
- **WHEN** se lee una orden que aún no tiene rider
- **THEN** `RiderID` es nil y `RiderAcceptedAt` es nil

#### Scenario: Orden con rider asignado
- **WHEN** se lee una orden que tiene rider asignado
- **THEN** `RiderID` contiene el UUID del rider y `RiderAcceptedAt` contiene el timestamp
