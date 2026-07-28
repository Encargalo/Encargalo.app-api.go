## ADDED Requirements

### Requirement: Gestión de catálogo por shop_id explícito
Un usuario con rol `Staff` SHALL poder gestionar el catálogo (categorías, productos, flavors, descuentos) de cualquier shop pasando el `shop_id` explícito en la URL, usando su propia sesión `Staff` — sin necesidad de emitir ni asumir ninguna otra sesión.

#### Scenario: Staff crea una categoría para un shop
- **WHEN** un usuario con rol `Staff` hace `POST /shops/staff/{shopId}/products/categories` con un body válido
- **THEN** el sistema crea la categoría asociada a `shopId`, igual que si la creara el owner desde `/business/products/categories`

#### Scenario: Staff lista el catálogo de un shop
- **WHEN** un usuario con rol `Staff` hace `GET /shops/staff/{shopId}/products/categories` (o `/macro-categories`, `/products`, `/categories/summary`)
- **THEN** el sistema retorna el catálogo de ese shop

### Requirement: Sin sesión suplantada
El sistema NO SHALL emitir ninguna sesión con una identidad distinta a la que inició sesión. Toda request de gestión de catálogo por staff SHALL quedar autenticada con la sesión `Staff` original (`session.UserID` es siempre el ID real del miembro de staff).

#### Scenario: Auditoría de una acción de staff
- **WHEN** un staff crea o edita un recurso del catálogo de un shop
- **THEN** el sistema puede loguear `staff_id` (de la sesión) y `shop_id` (del path) sin depender de ningún claim de sesión especial
