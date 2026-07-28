## ADDED Requirements

### Requirement: Estado de onboarding del shop
Todo `Shop` SHALL tener un campo `Status` con valores `draft` o `active`. Los shops creados por el flujo de staff SHALL nacer en `draft`. Los shops ya existentes antes de este change SHALL quedar migrados a `active`.

#### Scenario: Shop nuevo nace en draft
- **WHEN** el staff crea un shop nuevo
- **THEN** el shop queda persistido con `status = draft`

### Requirement: Creación de shop en borrador por staff
Un usuario con rol `Staff` SHALL poder crear un shop nuevo indicando al menos nombre y el `owner_id` de una cuenta ya vinculada como `Owner`. Los demás campos (logo, banner, dirección, coordenadas, teléfono) SHALL ser opcionales en la creación.

#### Scenario: Staff crea shop con datos mínimos
- **WHEN** un usuario con rol `Staff` envía nombre y owner_id válidos a la creación de shop
- **THEN** el sistema crea el shop en estado `draft` con esos datos, dejando el resto de campos vacíos

#### Scenario: Rol distinto de Staff intenta crear un shop
- **WHEN** un usuario sin rol `Staff` intenta crear un shop
- **THEN** el sistema responde con error de autorización y no crea el shop

### Requirement: Edición incremental del shop en borrador
Un usuario con rol `Staff` SHALL poder actualizar de forma parcial los campos de un shop en estado `draft` (nombre, slug, logo, banner, dirección, coordenadas, teléfono), en cualquier orden y sin exigir que todos los campos estén completos en una sola operación. Enviar un campo explícitamente en `null` SHALL limpiarlo; omitirlo SHALL dejarlo sin tocar.

#### Scenario: Staff completa el shop en varias llamadas
- **WHEN** el staff envía solo el logo en una actualización, y en una llamada posterior envía solo el banner
- **THEN** ambos campos quedan guardados en el shop sin afectar los demás campos ya cargados

#### Scenario: Staff limpia un campo con null explícito
- **WHEN** el staff envía `{"latitude": null}` sobre un shop que ya tenía coordenadas cargadas
- **THEN** la latitud queda vacía; los demás campos no se ven afectados

#### Scenario: Edición de un shop activo pero cerrado
- **WHEN** el staff intenta editar campos de un shop en estado `active` con `opened: false`
- **THEN** el sistema permite la edición igualmente (no bloquea correcciones posteriores a la activación)

### Requirement: Bloqueo de edición mientras el local está abierto
El sistema SHALL rechazar cualquier edición de los datos del shop (`PATCH /shops/staff/{id}`) mientras `opened` sea `true`. El catálogo de productos (`/shops/staff/{id}/products/*`) NO SHALL verse afectado por esta restricción.

#### Scenario: Staff intenta editar un local abierto
- **WHEN** el staff envía un `PATCH /shops/staff/{id}` sobre un shop con `opened: true`
- **THEN** el sistema rechaza la request con un error de conflicto y no modifica ningún campo

### Requirement: Activación manual del shop
Un usuario con rol `Staff` SHALL poder activar un shop que esté en estado `draft`, pasándolo a `active`, mediante una acción explícita separada de la edición de campos. La activación SHALL exigir el checklist completo: `slug`, `logo_image`, `banner`, `address`, `home_phone` y coordenadas (`latitude`/`longitude`).

#### Scenario: Staff activa un shop en draft
- **WHEN** el staff invoca la activación sobre un shop en estado `draft` con el checklist completo
- **THEN** el shop pasa a estado `active`

#### Scenario: Staff intenta activar un shop sin slug
- **WHEN** el staff invoca la activación sobre un shop en estado `draft` sin `slug` (u otro campo del checklist)
- **THEN** el sistema rechaza la activación indicando los campos faltantes

#### Scenario: Activar un shop que ya está activo
- **WHEN** el staff invoca la activación sobre un shop que ya está en `active`
- **THEN** el sistema no cambia el estado y responde de forma idempotente (sin error de negocio)

### Requirement: Bloqueo de apertura hasta activación
El sistema SHALL rechazar la apertura del local (`open: true` en el endpoint de apertura/cierre del owner) mientras el shop no esté en estado `active`.

#### Scenario: Owner intenta abrir un shop en draft
- **WHEN** el owner de un shop en estado `draft` intenta marcar `open: true`
- **THEN** el sistema responde con error indicando que el shop no está activado y no marca el shop como abierto

#### Scenario: Owner abre un shop activo
- **WHEN** el owner de un shop en estado `active` marca `open: true`
- **THEN** el sistema marca el shop como abierto, aplicando el comportamiento existente (promos de lanzamiento en la primera apertura)

### Requirement: Exclusión de shops en borrador de los listados públicos
Los listados y búsquedas de shops visibles para clientes (listado general, búsqueda, asignación Turbo) SHALL excluir los shops en estado `draft`.

#### Scenario: Cliente lista shops cercanos
- **WHEN** un cliente consulta el listado o búsqueda de shops
- **THEN** los shops en estado `draft` no aparecen en el resultado, aunque cumplan los demás criterios (tipo, radio, texto)
