## ADDED Requirements

### Requirement: Pedido Turbo exige cobertura 2 km
Al cotizar o crear un pedido cuyo `shop_id` corresponde a un shop `type = Turbo`, el sistema SHALL rechazar la operación si la distancia haversine entre el cliente (coords del draft) y el shop supera 2 km.

#### Scenario: Create dentro de cobertura
- **WHEN** el Customer crea un pedido a un Turbo a ≤2 km
- **THEN** el pedido se crea con ese `shop_id` y el flujo de estados/rider existente

#### Scenario: Create fuera de cobertura
- **WHEN** el Customer intenta crear/cotizar un pedido Turbo a >2 km
- **THEN** el sistema responde error de negocio (HTTP 400/422 según mapeo `errcustom`) y no crea la orden

---

### Requirement: ETA de shop Turbo sin prep fija
Para shops `type = Turbo`, el campo `time` (minutos) en respuestas de shop/entry SHALL calcularse como duración de manejo redondeada **sin** sumar los 20 minutos de preparación usados en Restaurant/Supermarket.

#### Scenario: Entry expone eta sin +20
- **WHEN** `GET /turbo/entry` asigna un nodo
- **THEN** `eta_minutes` / `shop.time` no incluyen el offset fijo de +20 de prep
