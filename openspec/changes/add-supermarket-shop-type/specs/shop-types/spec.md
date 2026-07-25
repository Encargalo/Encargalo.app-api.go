## ADDED Requirements

### Requirement: Tipos de tienda válidos
El sistema SHALL reconocer exactamente dos tipos de tienda en `business.shops.type`: `Restaurant` y `Supermarket`. Cualquier otro valor MUST ser rechazado por la restricción de base de datos y por la validación de dominio cuando aplique.

#### Scenario: Tienda restaurante válida
- **WHEN** una tienda tiene `type = "Restaurant"`
- **THEN** el sistema la acepta y la expone en `ShopResponse.type` como `"Restaurant"`

#### Scenario: Tienda supermercado válida
- **WHEN** una tienda tiene `type = "Supermarket"`
- **THEN** el sistema la acepta y la expone en `ShopResponse.type` como `"Supermarket"`

#### Scenario: Tipo inválido rechazado en BD
- **WHEN** se intenta persistir una tienda con `type` distinto de `Restaurant` o `Supermarket`
- **THEN** la base de datos rechaza la operación por violación del CHECK `shops_type_check`

---

### Requirement: Migración amplía el CHECK de tipo
El sistema SHALL incluir una migración goose que reemplace el CHECK actual (solo `Restaurant`) por uno que permita `Restaurant` y `Supermarket`, sin alterar las filas existentes.

#### Scenario: Tiendas existentes siguen siendo Restaurant
- **WHEN** se aplica la migración sobre una base con tiendas `type = 'Restaurant'`
- **THEN** todas las tiendas existentes conservan su tipo y el nuevo CHECK queda activo
