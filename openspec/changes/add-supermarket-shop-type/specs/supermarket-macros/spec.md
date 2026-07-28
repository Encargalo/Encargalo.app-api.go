## ADDED Requirements

### Requirement: Macro-categorías de supermercado sembradas
El sistema SHALL sembrar macro-categorías iniciales orientadas a supermercado en `products.macro_categories`, con slugs distintos de las macros de restaurante existentes, disponibles vía el listado de macros del negocio.

#### Scenario: Macros de súper visibles al Owner
- **WHEN** un usuario autenticado como Owner/OrderManager consulta `GET /api/v1/business/products/macro-categories`
- **THEN** la respuesta incluye las macros de supermercado sembradas (además de las de restaurante existentes)

#### Scenario: Slugs no colisionan con macros de comida
- **WHEN** se aplica la migración de seed de macros de supermercado
- **THEN** ningún slug nuevo duplica un slug ya existente (p. ej. no reutilizar `bebidas`; usar un slug diferenciado como `bebidas-super`)

---

### Requirement: Productos de supermercado reutilizan catálogo e availability
El sistema SHALL permitir que una tienda `Supermarket` registre productos (ítems) usando el flujo de catálogo existente, sin requerir sabores ni adiciones, y el Owner SHALL poder activar o desactivar un producto mediante `availability_status` (`active` | `hidden` | `out_of_stock`) en la actualización de ítem ya existente. No se introduce inventario numérico.

#### Scenario: Ítem plano de supermercado
- **WHEN** el Owner de una tienda `Supermarket` registra un ítem en una categoría con `selector_type = none` y sin adiciones
- **THEN** el ítem queda disponible en el catálogo del cliente según su `availability_status`

#### Scenario: Owner desactiva producto
- **WHEN** el Owner actualiza un ítem con `availability_status = "hidden"` (o `"out_of_stock"`)
- **THEN** el producto deja de ser pedible / visible según las reglas ya definidas de availability (sin stock numérico)

#### Scenario: Owner reactiva producto
- **WHEN** el Owner actualiza un ítem con `availability_status = "active"`
- **THEN** el producto vuelve a estar visible y pedible para el Customer (sujeto a reglas efectivas existentes)
