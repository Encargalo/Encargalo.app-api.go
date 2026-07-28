## Why

Hoy no existe ningún flujo para que el staff dé de alta un negocio nuevo: no hay endpoint para crear un shop, no hay forma de vincular una cuenta ya registrada (Customer) al rol Owner, y no hay ningún gate que impida abrir un local antes de que su registro esté completo. El plan operativo es que el staff (desde el CRM) cree el negocio, lo complete por partes (nombre, logo, banner, dirección, coordenadas) y cargue el catálogo inicial de productos — reutilizando el flujo de alta de producto que ya existe para el owner (imagen → Cloudinary → `public_id`/`hash`) en vez de construir un importador masivo nuevo. Mientras ese proceso no termine, el owner no debe poder abrir el local, porque abrir dispara las promos de lanzamiento (50% día 1, comisión 0% primeros 8 días).

## What Changes

- Nuevo campo `Status` (`draft` / `active`) en `Shop`, por defecto `draft`. `PATCH /business/shops/me/status` (apertura del local) rechaza `open: true` si el shop no está `active`.
- Nuevos endpoints de staff para gestionar el onboarding de un shop: crear en `draft`, actualizar campos de forma incremental (parcial, sin exigir completitud), y activar (`draft` → `active`) como acción manual explícita.
- Nuevo endpoint de staff para vincular una cuenta ya existente (buscada por teléfono) al rol `Owner` sobre un shop, insertando una fila en `users.user_roles` sin remover los roles que la cuenta ya tenga (p. ej. `Customer`).
- Nuevas rutas de staff para gestionar el catálogo de un shop directamente (categorías, productos, flavors, descuentos, macro-categorías), bajo `/shops/staff/{id}/products/*`. El staff sigue autenticado con su propia sesión `Staff` de siempre; el `shop_id` viaja explícito en la URL en vez de derivarse de una sesión suplantada. Sin sesión falsa, sin salto a otra app: el CRM puede construir estas pantallas dentro de su propio origen.

(Se descartó el enfoque inicial de sesión suplantada — ver Decisión #2 revisada en `design.md` — porque el hand-off de esa sesión entre el CRM y el Dashboard de Negocios, dos apps en orígenes distintos, dependía de que el navegador compartiera una cookie cross-site hacia el backend; en desarrollo (túnel) y potencialmente en cualquier navegador con bloqueo de cookies de terceros, esa cookie nunca llega y la sesión "impersonada" es inutilizable.)

## Capabilities

### New Capabilities
- `shop-staff-onboarding`: ciclo de vida de alta de un shop por staff — estado `draft`/`active`, creación, edición incremental, activación, y el gate que bloquea la apertura hasta estar activo.
- `owner-role-assignment`: vincular una cuenta existente al rol `Owner` de un shop sin afectar sus roles actuales.
- `staff-products-management`: rutas de staff para gestionar el catálogo (categorías, productos, flavors, descuentos) de cualquier shop pasando el `shop_id` explícito, sin sesión suplantada.

### Modified Capabilities
(ninguna — no hay specs archivadas todavía que describan comportamiento existente de shops/auth; el cambio a `SetOpenClose` se documenta como parte de `shop-staff-onboarding`)

## Impact

- `internal/shops/domain/models/shops/shops.go`: nuevo campo `Status` + migración goose.
- `internal/shops/domain/ports`, `app`, `repo`, `handler`: nuevos métodos de creación/edición/activación de shop por staff; `SetOpenClose` gana la validación de estado.
- `api/router/groups/shops.go`: nuevo grupo de rutas `/staff/shops` (rol `Staff`).
- `internal/users/repo/users.go` (o equivalente en `users`/`auth`): nuevo método para insertar fila en `users.user_roles` sin tocar las existentes.
- `internal/products/handler`: nuevo `StaffProductsHandler` que reusa `ports.ProductsApp` (sin cambios en `app`/`repo`/`ports` — ya reciben `shop_id` explícito) tomando el `shop_id` del path en vez de la sesión.
- `api/router/groups/`: nuevo grupo de rutas `/shops/staff/{id}/products/*` (rol `Staff`).
- Swagger + Postman (`postman/shops/`, `postman/products/`) actualizados para los endpoints nuevos.
- No afecta al Frontend de Negocios ni a la app de clientes. El CRM (`Encargalo.app-crm`, repo aparte) construye pantallas de gestión de catálogo dentro de su propio origen consumiendo estos endpoints — fuera del alcance de este change, pero ya no necesita redirigir a otra app.
