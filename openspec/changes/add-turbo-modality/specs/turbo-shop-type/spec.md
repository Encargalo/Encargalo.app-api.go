## ADDED Requirements

### Requirement: Tipo de tienda Turbo
El sistema SHALL reconocer `Turbo` como valor válido de `business.shops.type`, junto a `Restaurant` y `Supermarket`.

#### Scenario: Persistir shop Turbo
- **WHEN** se crea o actualiza una tienda con `type = "Turbo"`
- **THEN** la base de datos y el dominio lo aceptan y `ShopResponse.type` expone `"Turbo"`

#### Scenario: Tipo inválido sigue rechazado
- **WHEN** se intenta persistir un `type` distinto de `Restaurant`, `Supermarket` o `Turbo`
- **THEN** la BD rechaza la operación por el CHECK de tipo

---

### Requirement: Catálogo de nodo Turbo es solo productos listos
Un shop con `type = Turbo` SHALL operar con catálogo de productos listos (sin exigir sabores ni adiciones). El sistema no modela productos “turbo” dentro de un Restaurant/Supermarket en este cambio.

#### Scenario: Pedido turbo usa shop_id del nodo
- **WHEN** un Customer crea un pedido contra un shop `type = Turbo`
- **THEN** todos los ítems del pedido pertenecen a ese `shop_id` (sin mezcla con otros shops en el mismo draft turbo)
