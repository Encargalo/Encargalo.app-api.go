## MODIFIED Requirements

### Requirement: Visibilidad de códigos segmentada por rol
El sistema SHALL controlar la visibilidad de `pickup_code` y `delivery_code` según el rol del destinatario al enviar datos de una orden por WebSocket o API.

| Rol | `pickup_code` | `delivery_code` |
|---|---|---|
| Shop (OrderManager / Owner) | oculto | oculto |
| Customer | oculto | visible |
| Rider asignado | visible | oculto |

#### Scenario: Shop no recibe ningún código
- **WHEN** se emite un `order_update` a una conexión de tipo Shop
- **THEN** tanto `pickup_code` como `delivery_code` son `null` en el payload

#### Scenario: Customer recibe solo delivery_code
- **WHEN** se emite un `order_update` a una conexión de tipo Customer
- **THEN** `pickup_code` es `null` y `delivery_code` contiene el valor (si existe)

#### Scenario: Rider recibe solo pickup_code
- **WHEN** se emite un `order_update` o `order_accepted` a una conexión de tipo Rider
- **THEN** `pickup_code` contiene el valor (si existe) y `delivery_code` es `null`

#### Scenario: Mensaje new_order a riders disponibles no incluye códigos
- **WHEN** se emite `new_order` a riders en el pool de disponibles (sin orden asignada aún)
- **THEN** tanto `pickup_code` como `delivery_code` son `null`
