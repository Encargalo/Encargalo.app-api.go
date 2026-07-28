## Context

Hoy:
- No existe ningún endpoint para crear un `Shop`. Se insertan por fuera de la API.
- `Shop.LicenseStatus` (default `'active'`) ya existe pero significa "licencia vigente / suspendida por incumplimiento" — no "registro de onboarding completo". No se reutiliza para este propósito.
- `Shop.Opened` controla si el local está operando en el día a día; al pasar a `true` dispara las promos de lanzamiento (`FirstOpenedAt`, `LaunchPromoEndsAt`, `ZeroCommissionEndsAt`), vía `PATCH /business/shops/me/status` (`internal/shops/handler/shops.go`, `SetOpenClose`).
- El esquema de cuentas ya es multi-rol: `users.accounts` + `users.user_roles` (`user_id`, `role_id`, `shop_id`, `enabled`) + `users.roles`. `SearchAccountByPhone(ctx, phone, roles []string)` (`internal/users/repo/users.go`) ya hace el JOIN y devuelve `ShopID`/`RoleName` por esa vía. Agregarle un rol a una cuenta existente es insertar una fila; no requiere cambios de esquema.
- El catálogo (`POST /business/products/items|categories|flavors|discounts`, etc., en `api/router/groups/products.go`) ya implementa el flujo completo de imagen → Cloudinary → `public_id`/`hash` (`internal/pkg/uploadImage`). Nada de esto se toca. Los métodos de `app`/`repo`/`ports` de `products` ya reciben `shop_id` como parámetro explícito (DTOs con `ShopID`, o `shopID` como argumento directo) — solo el handler de negocio lo deriva hoy de `session.ShopID`.
- El rol `Staff` ya existe y ya tiene precedente de panel separado (refund-requests, payment-reviews en `orders`/`users`).

## Goals / Non-Goals

**Goals:**
- El staff puede crear un shop en estado `draft` y completarlo por partes (PATCH incremental, sin exigir completitud en cada paso).
- El staff puede vincular una cuenta ya existente (buscada por teléfono) al rol `Owner` de ese shop, sin remover los roles que la cuenta ya tenga.
- El staff puede cargar el catálogo inicial (categorías, productos, flavors, descuentos) de cualquier shop, pasando el `shop_id` explícito, con su propia sesión `Staff`.
- El owner no puede abrir el local (`open: true`) hasta que el shop esté `active`.
- La activación de `draft` → `active` es una acción manual y explícita del staff/admin.

**Non-Goals:**
- Importación masiva vía Excel/ZIP de productos e imágenes — descartada a favor de reutilizar el flujo de carga existente (rutas de staff con `shop_id` explícito).
- Un tercer estado de revisión (`pending_review`) separando "quien carga" de "quien aprueba" — se decide usar solo `draft`/`active`.
- Cambios al Frontend de Negocios o al CRM (repos aparte) — este change solo cubre el backend/API.
- Auditoría a nivel de columna en `products.items`/`categories` (ej. `created_by_staff_id`) — se resuelve con logging estructurado por ahora (ver Risks).
- Revertir un shop `active` de vuelta a `draft`, o suspenderlo — ya existe `LicenseStatus` para suspensión por incumplimiento; no se mezcla con este estado de onboarding.

## Decisions

**1. Reusar el flujo de carga existente (rutas de staff con `shop_id` explícito) en vez de import masivo vía Excel.**
El problema difícil de un import por Excel es la carga de imágenes: para conservar `public_id`/`hash` hace falta que cada imagen pase por Cloudinary igual que hoy, lo que exige un mecanismo de matching (Excel + ZIP de fotos por nombre de archivo) o un paso posterior de "subir foto por producto ya creado". Ambas opciones son pipelines nuevos con su propia superficie de bugs. Reutilizar la misma lógica de alta de producto (mismo `app`/`repo`, distinto punto de entrada HTTP) evita construir ese pipeline: cada producto se crea exactamente igual que lo haría el owner, con su imagen en el mismo paso.
*Alternativas consideradas*: (a) Excel + ZIP de imágenes emparejadas por nombre de archivo — más rápido para volúmenes grandes, pero requiere parser + validación + manejo de fallos parciales; (b) Excel solo de texto + carga de fotos asistida después — reduce el problema pero deja productos en estado "incompleto" mientras tanto. Se descartan ambas por ahora; quedan como posible mejora futura si el volumen real lo justifica.

**2. [REVISADA] Rutas Staff con `shop_id` explícito, en vez de sesión suplantada.**
La primera versión de esta decisión optaba por emitir una `ActiveSession` con el mismo shape que `SignInEmployee` (`role: Owner`, `shopID`) más `ActingStaffID`, para reutilizar el catálogo de rutas de negocio sin tocar ni un handler existente. En la práctica esto resultó frágil: el CRM (donde el staff inicia sesión) y el Dashboard de Negocios (donde vivían las rutas de catálogo) son dos SPAs en orígenes distintos; el hand-off dependía de que el navegador reenviara al Dashboard una cookie seteada por el backend en respuesta a la request del CRM — exactamente lo que el bloqueo de cookies de terceros (ya el comportamiento por defecto en navegadores modernos, y una política que Chrome está terminando de desplegar globalmente) impide. La sesión "impersonada" quedaba inutilizable fuera de configuraciones de navegador ad-hoc.

Se optó en cambio por que el staff gestione el catálogo con su propia sesión `Staff` de siempre, pasando el `shop_id` explícito en la URL (`/shops/staff/{id}/products/*`). El costo temido originalmente (tocar cada handler de negocio) resultó menor de lo esperado: `app`/`repo`/`ports` de `products` ya reciben `shop_id` como parámetro explícito en todos los métodos relevantes (los DTOs de creación ya traen `ShopID`, las búsquedas ya toman `shopID` como argumento) — el handler de negocio es la única capa que hoy lo deriva de `session.ShopID`. La superficie nueva es un handler paralelo (`StaffProductsHandler`) que toma `shopID` del path en vez de la sesión; `app`/`repo`/`ports` no cambian.
*Alternativa descartada (esta vez en firme)*: sesión suplantada cross-origen vía cookie — descartada por depender de un comportamiento del navegador que ya no es confiable ni en desarrollo ni, a mediano plazo, en producción.

**3. Estados del shop: solo `draft`/`active`.**
Cubre exactamente el caso descrito (bloquear apertura hasta que el registro esté listo) sin introducir un flujo de aprobación en dos manos. Si más adelante se necesita separar "quien carga" de "quien aprueba", se puede insertar `pending_review` entre ambos sin romper el contrato (`active` sigue siendo el único estado que habilita `Opened`).

**4. Multi-rol sobre la cuenta existente, no cuentas separadas.**
El esquema `users.user_roles` ya soporta esto sin cambios. Una misma persona puede seguir siendo `Customer` (comprar en la app) y ser `Owner` de su propio negocio con la misma cuenta/teléfono.

**5. [OBSOLETA] Scope de sesión suplantada vía middleware.**
Ya no aplica — no hay sesión suplantada que acotar (ver Decisión #2 revisada). El scope de lo que el staff puede hacer queda dado simplemente por qué rutas existen bajo `/shops/staff/{id}/products/*` (solo catálogo) y por el `Require(models.Staff)` de siempre; no hace falta ningún middleware ni TTL especial, la sesión del staff es una sesión `Staff` normal.

## Risks / Trade-offs

- **[Riesgo]** Las tablas `products.items`/`products.categories` no tienen columna de autoría — una fila creada por el staff es indistinguible en DB de una creada por el owner real.
  → **Mitigación**: loguear `staff_id` (de `session.UserID`) + `shop_id` (del path) + ruta + método en cada request de staff sobre catálogo (log estructurado, reusa el logging de request existente). Si a futuro se requiere trazabilidad a nivel de fila, es un change aparte (columna `created_by_staff_id`).
- **[Riesgo]** Un `Shop.Status` con default `draft` puede romper código existente que asume que todo shop insertado está operativo (listados de `/shops/all`, `/shops/search`, `ListTurboCandidates`).
  → **Mitigación**: esos queries deben excluir `status != 'active'` igual que ya excluyen por `license_status`; se cubre explícitamente en tasks.
- **[Riesgo]** El staff vincula el rol `Owner` a la cuenta equivocada por una búsqueda de teléfono ambigua o errónea.
  → **Mitigación**: el endpoint de vinculación devuelve los datos de la cuenta encontrada (nombre, teléfono) para confirmación explícita antes de asignar el rol; solo accesible con rol `Staff`, quedando auditado por la sesión misma.
- **[Riesgo]** Migración de `Status` sobre shops ya existentes en producción: si el default de columna fuera `draft`, todo shop ya operativo quedaría bloqueado para reabrir.
  → **Mitigación**: la migración goose agrega la columna con default `'draft'` pero hace `UPDATE` explícito a `'active'` sobre todas las filas existentes en el mismo `Up`; solo los shops creados después de este change (vía el nuevo flujo) nacen en `draft`.

## Migration Plan

1. Migración goose: agregar `status varchar default 'draft'` a `business.shops`; `UPDATE business.shops SET status = 'active'` (backfill de lo ya existente); `Down` elimina la columna.
2. Deploy del backend con los nuevos endpoints y el gate en `SetOpenClose` (no rompe nada porque todo shop preexistente ya quedó `active` por el backfill).
3. El CRM (repo aparte) se integra contra estos endpoints en un change propio, fuera de este backend.

## Open Questions

- ¿La activación (`draft` → `active`) valida automáticamente campos mínimos (slug, logo, banner, dirección, coordenadas) antes de aceptar el `PATCH`, o confía en el juicio del staff sin validación de completitud? (se propone como tarea validar solo los campos obligatorios de siempre — no una regla nueva de "mínimo N productos"). Resuelto: sí se valida (`Shop.Completion()`), y `slug` (antes `tag`) es parte del checklist obligatorio.
