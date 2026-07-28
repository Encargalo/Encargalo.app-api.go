# Guía Frontend — Tipos de tienda (Restaurant / Supermarket)

Contrato para la **app cliente** (filtro por tipo) y el **panel de negocios** (catálogo de supermercado).

Auth: cookie `encargalo-session`.  
Base URL local: `http://localhost:3000`.

Cambio backend: OpenSpec `add-supermarket-shop-type`.

---

## 1. Idea de producto

La plataforma ahora admite dos tipos de tienda:

| Valor API (`type`) | Uso |
|---|---|
| `Restaurant` | Comida rápida (sabores / adiciones opcionales) |
| `Supermarket` | Productos “planos” (arroz, pollo, etc.) sin customización |

**No hay** `Pharmacy` en este release.  
**No hay** inventario numérico: el owner activa/desactiva con `availability_status`.

---

## 2. App cliente — filtrar tiendas

### Endpoints

| Método | Ruta | Rol |
|---|---|---|
| `GET` | `/shops/all` | `Customer` |
| `GET` | `/shops/search` | `Customer` |

### Query params

| Param | Requerido | Valores | Comportamiento |
|---|---|---|---|
| `lat` | sí | number | Latitud |
| `lon` | sí | number | Longitud |
| `q` | solo en `/search` | string | Texto de búsqueda |
| `type` | **no** | `Restaurant` \| `Supermarket` | Filtra por tipo. **Si se omite → ambos** |

`type` inválido (ej. `Pharmacy`) → **HTTP 400**.

### Ejemplos

```http
GET /shops/all?lat=4.678034&lon=-74.0496399&type=Restaurant
GET /shops/all?lat=4.678034&lon=-74.0496399&type=Supermarket
GET /shops/all?lat=4.678034&lon=-74.0496399

GET /shops/search?q=arroz&lat=4.678034&lon=-74.0496399&type=Supermarket
GET /shops/search?q=pizza&lat=4.678034&lon=-74.0496399&type=Restaurant
GET /shops/search?q=cola&lat=4.678034&lon=-74.0496399
```

### UX sugerida

```
[ Restaurantes ]  [ Supermercado ]  [ Todos ]
       │                  │              │
 type=Restaurant   type=Supermarket   (sin type)
```

### Response

Cada tienda ya incluye `type`. Tipar en front:

```ts
type ShopType = 'Restaurant' | 'Supermarket'

interface ShopResponse {
  id: string
  name: string
  tag: string
  // ...
  type: ShopType
  // ...
}
```

Usar `shop.type` para badges, iconos o navegación al catálogo (p. ej. ocultar UI de sabores/adiciones si es `Supermarket`).

### Pedido / catálogo dentro de una tienda

Sin cambios de contrato en órdenes. Un producto de súper se pide como ítem normal **sin** `flavors` ni `additions`.

---

## 3. Panel de negocios — qué cambia

### 3.1 `GET /business/shops/me`

La respuesta incluye `type`. El front de negocios **debe** tiparlo y usarlo:

```ts
type ShopType = 'Restaurant' | 'Supermarket'

interface ShopData {
  // ...campos existentes
  type: ShopType
}
```

| Si `type === 'Supermarket'` | Recomendación UX |
|---|---|
| Sabores / selector | Ocultar o no exigir (`selector_type: 'none'`) |
| Adiciones | Ocultar asignación de adiciones |
| Macros al crear categoría | Preferir macros de súper (ver abajo) |
| Activar / desactivar producto | Usar `availability_status` (ya existe) |

Hoy el tipo de la tienda lo define ops/DB (`Restaurant` o `Supermarket`). El panel no tiene endpoint público de “cambiar tipo” en este cambio.

### 3.2 Macros de supermercado

`GET /business/products/macro-categories` ahora también lista macros de súper (además de las de comida):

| Name | Slug |
|---|---|
| Abarrotes | `abarrotes` |
| Frescos | `frescos` |
| Lácteos | `lacteos` |
| Bebidas Súper | `bebidas-super` |
| Aseo | `aseo` |
| Otros Súper | `otros-super` |

Nota: el listado **aún mezcla** macros de restaurant y súper. Filtrar en UI según `shopData.type` si queréis evitar que un súper elija “Hamburguesas”.

### 3.3 Productos de supermercado (ítems planos)

Al crear categoría:

```json
{
  "name": "Abarrotes",
  "macro_category_id": "<uuid de abarrotes>",
  "selector_type": "none"
}
```

Al crear ítem: **sin** `flavor_ids` / `addition_ids` (o arrays vacíos). Mismos campos de siempre: `name`, `price`, `image`, `description`, `category_id`.

### 3.4 Activar / desactivar producto (sin inventario)

`PUT /business/products/{item_id}` con:

```json
{
  "name": "Arroz 1kg",
  "price": 250,
  "description": "Arroz blanco",
  "availability_status": "hidden"
}
```

| `availability_status` | Efecto para el cliente |
|---|---|
| `active` | Visible y pedible |
| `out_of_stock` | Visible, no pedible |
| `hidden` | No aparece en el catálogo del cliente |

Para un toggle simple “activo/inactivo” en súper: `active` ↔ `hidden` (o `out_of_stock` si queréis que se vea “agotado”).

---

## 4. Checklist por app

### App cliente

- [ ] Tipar `ShopType` y el campo `type` en respuestas de shops
- [ ] Tabs / filtro: `Restaurant` | `Supermarket` | Todos (sin param)
- [ ] Pasar `type` en `/shops/all` y `/shops/search`
- [ ] Manejar 400 si `type` inválido
- [ ] En detalle de tienda `Supermarket`, no forzar UI de sabores/adiciones

### Panel negocios

- [ ] Leer `type` de `/business/shops/me`
- [ ] Si `Supermarket`: UX de catálogo plano (sin sabores/adiciones)
- [ ] Preferir / filtrar macros de súper al crear categorías
- [ ] Toggle producto con `availability_status` (`active` / `hidden` o `out_of_stock`)

---

## 5. Fuera de alcance (no implementar en front aún)

- Tipo `Pharmacy`
- Stock numérico / barcode / SKU / peso variable
- Endpoint nuevo solo para “toggle” (usar update de ítem)
- Filtrado de macros por tipo en el backend (hoy es responsabilidad del front si se desea)

---

## 6. Referencias

- OpenSpec: `openspec/changes/add-supermarket-shop-type/`
- Postman: `postman/shops/shops.postman_collection.json`
- Swagger: `GET /shops/all`, `GET /shops/search` (param `type`)
