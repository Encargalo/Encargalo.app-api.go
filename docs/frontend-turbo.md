# Guía Frontend — Turbo (estilo Rappi)

Contrato para la **app cliente**.  
Auth: cookie `encargalo-session` (rol `Customer`).  
Base URL local: `http://localhost:3000`.

OpenSpec: `openspec/changes/add-turbo-modality/`.

---

## 1. Idea de producto

Turbo es un **vertical del home** (tile), no un local que el usuario elige de una lista.

```
Home
├── [Restaurantes]     → /shops/all?type=Restaurant
├── [Supermercado]     → /shops/all?type=Supermarket
└── [Turbo · 10 min]   → GET /turbo/entry  ★
         │
         ▼
   Catálogo del nodo asignado (shop.name = "Turbo Fresh", etc.)
         │
         ▼
   Carrito / Checkout (muestran ese name)
         │
         ▼
   Pedido normal (shop_id del nodo → almacén + rider pickup)
```

Detrás siempre hay un **shop real** (`type=Turbo`, owner propio, coords).  
El backend **auto-asigna** el más cercano ≤ **2 km**.

---

## 2. Backend — entry

### `GET /turbo/entry?lat=&lon=`

| Campo | Tipo | Notas |
|---|---|---|
| `available` | bool | Si no hay nodo → `false` (ocultar tile o empty state) |
| `shop` | `ShopResponse` | Nodo asignado; usar `shop.id` y `shop.name` |
| `eta_minutes` | number | Sin +20 de prep |
| `coverage_km` | number | `2` |

Ejemplo disponible:

```json
{
  "available": true,
  "shop": {
    "id": "…",
    "name": "Turbo Fresh",
    "type": "Turbo",
    "opened": true,
    "time": 8,
    "delivery_fee": 1.2
  },
  "eta_minutes": 8,
  "coverage_km": 2
}
```

No disponible:

```json
{ "available": false }
```

---

## 3. Flujo de pantallas (como Rappi)

### 3.1 Home

- Mostrar tile **Turbo** solo si `available === true` (o mostrar tile y empty state si false).
- Badge marketing: “10 min” (copy); ETA real = `eta_minutes`.
- **No** meter shops Turbo en el feed general de restaurantes/súper.
  - `GET /shops/all` sin `type` → Restaurant + Supermarket (3 km), **sin** Turbo.
  - Turbo solo vía entry (o `type=Turbo` si necesitás debug).

### 3.2 Al tocar Turbo

1. `GET /turbo/entry?lat=&lon=`
2. Guardar en estado de sesión turbo: `{ shopId, shopName, etaMinutes }`
3. Navegar a catálogo: reutilizar `GET /products?shop_id=<shop.id>` (o el path actual de productos por shop)
4. Header del catálogo: marca “Turbo” + **`shop.name`** (ej. “Turbo Fresh”)

### 3.3 Carrito

- Título / store row: **`shop.name`** (como “Turbo Fresh” en Rappi).
- Solo ítems de ese `shopId`.
- CTA vaciar canasta, mínimo de compra si aplica (reglas de producto aparte).

### 3.4 Checkout / pagar

- Resumen del pedido agrupado bajo **`shop.name`**.
- Badge opcional “Turbo” / “Express”.
- Crear orden con el **mismo** `shop_id` (endpoints de orders existentes).

### 3.5 Cambio de dirección

```
Si cambia lat/lon:
  volver a GET /turbo/entry
  si !available → vaciar carrito turbo + mensaje “no hay Turbo en tu zona”
  si shop.id distinto → vaciar carrito + reasignar al nuevo nodo
  si mismo shop.id → mantener carrito
```

---

## 4. Tipado sugerido

```ts
type ShopType = 'Restaurant' | 'Supermarket' | 'Turbo'

interface TurboEntry {
  available: boolean
  shop?: ShopResponse  // presente si available
  eta_minutes?: number
  coverage_km?: number
}
```

---

## 5. Panel negocios (Owners Turbo)

Sin flujo especial de “marca Turbo” en MVP:

- El owner opera su shop como hoy (`/business/shops/me`).
- `type` del shop debe ser `Turbo` (ops/onboarding).
- El **`name`** que configure es lo que ve el cliente en carrito (“Turbo Fresh”, “Turbo Chacao”, …).
- Catálogo: productos listos, `selector_type: none`, sin adiciones.
- Activar/desactivar: `availability_status` (`active` | `hidden` | `out_of_stock`).

---

## 6. Checklist app cliente

- [ ] Tile Turbo en home (prioridad visual junto a Restaurantes / Súper)
- [ ] Llamar `/turbo/entry` con coords de la dirección activa
- [ ] Ocultar / empty state si `available=false`
- [ ] Catálogo con `shop.id` asignado
- [ ] Carrito y checkout muestran `shop.name`
- [ ] No listar Turbo en feed general de shops
- [ ] Re-entry al cambiar dirección; vaciar carrito si cambia el nodo
- [ ] Copy “10 min” + ETA real de API

---

## 7. Fuera de alcance (no bloquear MVP)

- Prioridad de riders / SLA real garantizado
- Inventario numérico
- Un carrito con varios almacenes Turbo
- Fee especial Turbo (default: misma fórmula de envío)

---

## 8. Referencias

- OpenSpec: `openspec/changes/add-turbo-modality/`
- Súper / tipos: `docs/frontend-supermarket-shop-type.md`
- Postman: añadir `GET /turbo/entry` cuando exista el endpoint
