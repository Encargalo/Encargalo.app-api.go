## Context

Hoy `business.shops.type` tiene un CHECK que solo permite `'Restaurant'`. Los endpoints customer `GET /shops/all` y `GET /shops/search` no filtran por tipo; `ShopResponse.type` ya se serializa pero es siempre `Restaurant`. El catálogo de productos ya soporta ítems sin sabores/adiciones y `availability_status` (`active` | `hidden` | `out_of_stock`) para que el Owner active/desactive productos — no hace falta inventario.

Este cambio abre el tipo `Supermarket` y expone un filtro opcional `type` al Customer, más macros iniciales de súper.

## Goals / Non-Goals

**Goals:**

- Permitir tiendas `Restaurant` y `Supermarket` a nivel de BD y dominio.
- Filtrar listado/búsqueda de tiendas por `type` opcional (omitido = ambos).
- Sembrar macro-categorías útiles para catálogo de supermercado.
- Documentar para el front el contrato del query param y los valores válidos.
- Reutilizar availability existente para activar/desactivar productos (sin código nuevo de inventario).

**Non-Goals:**

- Tipo `Pharmacy` u otros verticales.
- Inventario numérico, SKU, barcode, unidad de medida, peso variable.
- Endpoint nuevo de toggle de producto (ya existe vía update item + `availability_status`).
- Filtrar macros por tipo de tienda en `GET /business/products/macro-categories` (puede venir después; el Owner elige macros al crear categorías).
- Cambiar el flujo de pedidos, comisiones o Redis Streams.
- Alta pública de tiendas vía API (hoy no hay registro de shop en customer/business routes revisadas; el tipo se asigna al crear el registro en BD / proceso).

## Decisions

### 1. Valores de tipo: strings canónicos `Restaurant` | `Supermarket`

- **Decisión:** Mantener el casing existente (`Restaurant`) y añadir `Supermarket` (mismo estilo PascalCase).
- **Alternativas:** slug lowercase (`restaurant`/`supermarket`) — descartado por **BREAKING** en datos y respuestas actuales.
- **Rationale:** Cero migración de filas existentes; solo ampliar el CHECK.

### 2. Filtro `type` opcional en query, validado en DTO

- **Decisión:**
  - Añadir `Type string \`query:"type" validate:"omitempty,oneof=Restaurant Supermarket"\`` a:
    - `SearchShopsQuery` (para `/shops/search`)
    - Un DTO de listado (p. ej. extender o componer con `Coords` para `/shops/all` — preferible un `ListShopsQuery` con `lat`, `lon`, `type` opcional para no contaminar `Coords` reutilizado en `GetShopsBy`).
  - Si `type` vacío → no aplicar `WHERE` de tipo.
  - Si `type` inválido → HTTP 400.
- **Alternativa:** `type` obligatorio — descartado por decisión de producto (sin param = ambos).

### 3. Filtrado en repo (SQL), no en memoria

- **Decisión:** `WHERE s.type = ?` cuando el filtro viene informado, en `GetAllShops` y `SearchShopCandidates`.
- **Rationale:** Escala con más tiendas; índice opcional `(type)` si el volumen lo justifica (no obligatorio en MVP).

### 4. Migración: drop + recreate CHECK + seed macros

- **Decisión:** Migración goose `00051_shop_type_supermarket_and_macros.sql` (ajustar número si ya existe 00051):
  1. `ALTER TABLE business.shops DROP CONSTRAINT shops_type_check;`
  2. `ADD CONSTRAINT shops_type_check CHECK (type IN ('Restaurant', 'Supermarket'));`
  3. `INSERT` macros supermercado (slugs nuevos, IDs fijos estilo seed actual).
- **Macros iniciales sugeridas:** `abarrotes`, `frescos`, `lacteos`, `bebidas-super`, `aseo`, `otros-super` (nombres display en español). Evitar colisión con slug `bebidas` de restaurant.
- **Down:** restaurar CHECK solo `Restaurant` (falla si existen filas `Supermarket`) + borrar macros sembradas.

### 5. Dominio Go: constantes `ShopType`

- **Decisión:** Archivo `internal/shops/domain/models/shops/shopType.go` con:
  - `ShopTypeRestaurant = "Restaurant"`
  - `ShopTypeSupermarket = "Supermarket"`
  - `IsValidShopType(string) bool`
- Sin providers DI nuevos (sin módulos nuevos).

### 6. Productos y availability: sin cambios de modelo

- **Decisión:** No tocar el modelo de sabores/adiciones. Un producto de súper es un `Item` con categoría `selector_type=none`, sin `flavor_ids`/`addition_ids`. Activar/desactivar = `UpdateItem` con `availability_status`.
- **Rationale:** Cumple el requisito del Owner sin inventario.

### 7. Errores

- Validación de query inválido → 400 vía `validate` del DTO (mismo patrón actual).
- No se requieren nuevos sentinel en `errcustom` salvo que se introduzca escritura de tipo en algún comando futuro.

### 8. DI / eventos

- Sin providers nuevos en `di.go`.
- Sin impacto en Redis Stream ni comisión CreditManager.

## Risks / Trade-offs

- **[Riesgo] Macros de comida y súper en el mismo listado business** → Owner de súper podría asignar macro `hamburguesas`. **Mitigación:** documentar convención; fase 2 filtrar macros por `shop.type` de la sesión.
- **[Riesgo] Tiendas `Supermarket` creadas solo por ops/DB** → OK para MVP; si después hay API de alta, debe validar `IsValidShopType`.
- **[Riesgo] Front asume type siempre presente y solo Restaurant** → `type` ya está en `ShopResponse`; documentar valores nuevos en Swagger/Postman.
- **[Trade-off] Sin filtrar trending/best-sellers por tipo de tienda** → Fuera de alcance; el entry point es listado/búsqueda de shops. Se puede añadir después si el home muestra productos globales.

## Migration Plan

1. Desplegar migración `00051_...` (CHECK + seed macros).
2. Desplegar API con filtro `type`.
3. Ops puede marcar/crear tiendas `Supermarket` y categorías con macros nuevas.
4. Front envía `type` en tabs; sin param mantiene comportamiento “todas”.

**Rollback:** revertir API primero (filtro ignorado/ausente); luego goose down solo si no hay tiendas `Supermarket` en producción.

## Open Questions

- Ninguna bloqueante. Detalle fino de nombres/slugs de macros puede ajustarse en la migración sin cambiar el contrato del filtro `type`.
