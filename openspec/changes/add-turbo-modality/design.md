## Context

Encargalo ya lista shops por proximidad (3 km), calcula fee/ETA con Google + prep fija (+20), y acaba de introducir `Restaurant` | `Supermarket` con filtro `type`. Turbo (estilo Rappi) añade un vertical de home: el cliente no elige entre N turbos; el backend asigna **un** nodo `type=Turbo` ≤2 km y el front muestra su `name` en catálogo/carrito/checkout (como “Turbo Fresh”).

Cada nodo = shop real, owner distinto, catálogo solo de productos listos.

## Goals / Non-Goals

**Goals:**

- UX Rappi: tile Turbo → catálogo del nodo asignado → carrito con nombre del local.
- Auto-asignación por geo (≤2 km, abierto, license active).
- Reutilizar catálogo (`/products?shop_id=`), órdenes, WS, rider pickup (coords del shop).
- ETA sin +20 prep para Turbo; radio 2 km en discovery y en create/quote.
- Extender `type` con `Turbo`.

**Non-Goals:**

- SLA operativo garantizado / prioridad rider (fase 2).
- Inventario, barcode, stock multi-nodo, carrito multi-almacén.
- Shop fantasma “Encargalo Turbo” sin owner.
- Mezclar productos turbo dentro de un Restaurant/Supermarket.
- Cambiar comisión CreditManager ni streams salvo reutilizar el flujo actual.

## Decisions

### 1. Turbo = `shops.type`, no flag por ítem

- Un shop Turbo solo vende productos listos (convención: `selector_type=none`, sin adiciones).
- Mezcla imposible a nivel carrito: todo el catálogo del nodo es turbo.

### 2. Discovery estilo Rappi: entry + asignación

```
GET /api/v1/turbo/entry?lat=&lon=
→ 200 {
    available: true,
    shop: ShopResponse,           // nodo asignado (name visible = "Turbo Fresh")
    eta_minutes: <driving>,       // sin +20
    coverage_km: 2
  }
→ 200 { available: false }        // no hay nodo ≤2 km abierto
```

- Criterio MVP: **más cercano** (haversine), `opened=true`, `license_status=active`, `type=Turbo`, distancia ≤ 2 km.
- Empate: mayor `score`, luego `id`.
- El front **no** lista turbos en el feed de restaurantes como paso obligatorio; el tile llama a entry.

### 3. Radios por tipo

| Tipo | Radio listado / pedido |
|---|---|
| Restaurant / Supermarket | 3 km (actual) |
| Turbo | **2 km** |

- `GET /shops/all?type=Turbo` y search con `type=Turbo` usan radio 2 km (útil ops/debug; home usa entry).
- Quote/create: si `shop.type=Turbo`, rechazar si distancia (haversine o road — **MVP haversine coherente con listado**) > 2 km.

### 4. ETA

- Turbo: `time = ceil(driving_duration_minutes)` (sin +20).
- Marketing copy “10 min” / “máx. 10 min” es front; el API puede exponer `eta_minutes` y opcional `promise_minutes: 10`.

### 5. Órdenes

- Sin cambio de máquina de estados.
- `shop_id` / `batch_id` del nodo asignado.
- Rider pickup = address/coords del shop (ya en el modelo).
- Owner/OrderManager del nodo ven la orden en WS como hoy.

### 6. Cambio de dirección del cliente

- Si el usuario cambia `lat/lon` y el nodo asignado queda fuera de 2 km → entry de nuevo; front **vacía carrito** turbo o reasigna solo si el nuevo nodo es el mismo `shop_id`.
- MVP: vaciar / forzar re-entry si cambia dirección.

### 7. Relación con supermarket change

- Depende de (o continúa) el CHECK multi-tipo: `Restaurant | Supermarket | Turbo`.
- Validación query `type` incluye `Turbo`.
- Si `add-supermarket-shop-type` ya está mergeado, migración Turbo solo añade el valor; si no, una migración unifica los tres.

### 8. DI / módulos

- Preferible métodos en `shops` app/repo (`ResolveTurboNode`, `GetTurboEntry`) + grupo router `/turbo`.
- Sin bounded context nuevo obligatorio en MVP.

## Risks / Trade-offs

- **[Riesgo] Nodo más cercano sin stock** → Mitigación fase 2: fallback al siguiente; MVP availability por ítem.
- **[Riesgo] Cliente cree que “Turbo” es un solo almacén global** → Mostrar `shop.name` en carrito/checkout (como Rappi “Turbo Fresh”).
- **[Riesgo] Radio haversine vs road** → Shop ≤2 km aéreo pero lejos en calle; aceptar en MVP o usar Distance Matrix después.
- **[Trade-off] Sin prioridad rider** → La promesa 10 min es comercial hasta fase 2 operativa.

## Migration Plan

1. Ampliar CHECK `type` con `Turbo`.
2. Deploy API entry + radio 2 km + guards de orden.
3. Onboard N owners con shops `type=Turbo` (nombres distintos por zona).
4. Front: tile + flujo entry.
5. Rollback: quitar tile; shops Turbo dejan de asignarse si se revierte entry (o se apagan `opened`).

## Open Questions

- Fee Turbo: ¿misma fórmula por km o envío fijo/promo? (default MVP: misma fórmula).
- ¿Mostrar co-branding partner además de `name`? (opcional; no bloquea).
