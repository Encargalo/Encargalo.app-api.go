## Why

La plataforma solo admite tiendas de tipo `Restaurant` (comida rápida con sabores y adiciones). Para vender productos de supermercado (arroz, pollo, etc.) sin customización, el backend debe reconocer el tipo `Supermarket` y permitir al cliente filtrar tiendas por tipo. Sin esto el front no puede separar restaurantes de súper.

## What Changes

- Ampliar `business.shops.type` para aceptar `Restaurant` y `Supermarket` (sin farmacia en este cambio).
- Filtro opcional `type` en listado y búsqueda de tiendas para el Customer:
  - `type=Restaurant` → solo restaurantes
  - `type=Supermarket` → solo supermercados
  - omitido → ambos tipos
- Semilla de macro-categorías orientadas a supermercado (taxonomía inicial simple).
- Reutilizar el catálogo de productos existente: ítems planos sin sabores/adiciones; activar/desactivar vía `availability_status` ya existente (`active` | `hidden` | `out_of_stock`).
- **No** se introduce inventario numérico, barcode, SKU, peso variable ni tipo `Pharmacy`.

## Capabilities

### New Capabilities

- `shop-types`: Tipos de tienda `Restaurant` y `Supermarket` en dominio y BD; invariantes de valores válidos.
- `shop-type-filter`: Query opcional `type` en endpoints customer de listado/búsqueda de tiendas.
- `supermarket-macros`: Macro-categorías iniciales para catálogo de supermercado.

### Modified Capabilities

- (ninguna — no hay specs principales previas de shops/products en `openspec/specs/` para este alcance)

## Impact

- Migración goose: relajar/ampliar `shops_type_check` en `business.shops`; seed de macros de supermercado.
- Dominio `shops`: constantes/validación de `ShopType`; DTOs de query (`SearchShopsQuery`, `Coords`/listado) con `type` opcional.
- Repo `shops`: `WHERE type = ?` cuando el filtro viene informado.
- Handlers customer: `GET /shops/all`, `GET /shops/search` (y Swagger/Postman).
- Dominio `products`: seed macros; sin cambio obligatorio al modelo de ítems (ya soporta sin sabores/adiciones y availability).
- Alta/actualización de tiendas (si existe o se toca): aceptar `Supermarket` además de `Restaurant`.
- Sin impacto en Redis Streams ni comisión CreditManager (ciclo de pedidos sin cambios).
