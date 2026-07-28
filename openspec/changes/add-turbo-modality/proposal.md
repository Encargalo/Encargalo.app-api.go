## Why

Los clientes necesitan pedidos ultra-rápidos (≤10 min) de productos listos (refresco, arroz, etc.) cuando están cerca. El patrón Rappi — tile “Turbo” en home, catálogo unificado sin elegir local, nombre del nodo en carrito/checkout — convierte Turbo en una modalidad de discovery con auto-asignación geográfica, no en un shop fantasma ni en un flag por producto.

## What Changes

- Nuevo valor de tienda `type = Turbo` (además de `Restaurant` y `Supermarket`).
- Cada nodo Turbo es un `Shop` real con owner distinto, coords, catálogo propio (solo productos listos; sin mezclar con restaurant/súper en el mismo shop).
- Radio de cobertura Turbo: **2 km** (haversine); el resto de tipos sigue en 3 km.
- Endpoint de entrada estilo Rappi: dado `lat`/`lon`, asigna el Turbo abierto más cercano (o elegible) y expone su `shop_id` + `name` (ej. “Turbo Fresh”).
- ETA Turbo sin el +20 min de prep (driving duration, copy marketing ~10 min).
- Validación en quote/create: pedidos a shops Turbo exigen distancia ≤2 km.
- Front cliente: tile Turbo → catálogo del shop asignado → carrito/checkout muestran `shop.name`; no listar turbos como feed de restaurantes.
- **No** inventario numérico, **no** prioridad rider SLA real en MVP, **no** catálogo global multi-almacén en un solo carrito.

## Capabilities

### New Capabilities

- `turbo-shop-type`: Tipo de tienda `Turbo` en dominio/BD e invariantes de catálogo (ítems planos).
- `turbo-node-assignment`: Asignación automática del nodo Turbo por geo (≤2 km) para entry/home.
- `turbo-order-rules`: Reglas de distancia/ETA en quote y create para shops Turbo.
- `turbo-frontend-contract`: Contrato UX estilo Rappi (tile, catálogo, carrito, checkout) documentado para apps cliente.

### Modified Capabilities

- `shop-type-filter` (del change `add-supermarket-shop-type`): ampliar `oneof` / CHECK para incluir `Turbo` y radio distinto cuando `type=Turbo`.

## Impact

- Migración: ampliar `shops_type_check` con `Turbo`; constante radio 2 km.
- `shops`: dominio `ShopTypeTurbo`, entry/assignment API, filtro listado con radio por tipo.
- `orders`: guard de distancia para Turbo en quote/create; ETA display sin prep fija.
- `products`: sin modelo nuevo obligatorio; macros/categorías de súper pueden reutilizarse en nodos Turbo.
- Docs: `docs/frontend-turbo.md` + Postman/Swagger.
- Panel negocios: un Owner de Turbo opera su shop como hoy (`/business/shops/me`); el `name` visible al cliente es el del nodo.
