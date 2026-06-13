# Spec: Buscador de tiendas y productos por proximidad

## Problema

El app no tiene búsqueda textual. El usuario necesita poder escribir "pizza" y obtener tiendas que venden pizza aunque la palabra "pizza" no aparezca en el nombre de la tienda, o escribir "papa" y encontrar "Papajons" por coincidencia parcial. Los resultados se limitan a un radio de 3 km.

---

## Estado actual del backend

| Área | Estado |
|------|--------|
| Coordenadas de tiendas | `latitude` / `longitude` como DOUBLE PRECISION en `business.shops` |
| Geo-queries | Haversine en SQL raw (sin PostGIS) |
| Búsqueda de tiendas | Solo por `tag` o `id` (match exacto) |
| Búsqueda de productos | Solo por `shop_id`, `category_id`, `item_id` |
| Full-text search | **No existe** |
| Índices en name / description | **No existen** |

---

## Solución: PostgreSQL Full-Text Search + pg_trgm

### Qué se activa

- Extensión `pg_trgm` — ya viene incluida en Postgres, solo hace falta `CREATE EXTENSION`.
- Índices GIN sobre las columnas de texto relevantes para que las queries sean rápidas.

### Cómo resuelve cada caso de búsqueda

La query combina dos mecanismos con `OR`:

| Mecanismo | Qué cubre | Ejemplo |
|-----------|-----------|---------|
| `tsvector @@ tsquery` | Palabras completas con stemming y acentos en español | "pizzas" → encuentra "pizza", "café" → "cafe" |
| `ILIKE '%query%'` | Substrings parciales, cualquier posición | "papa" → "Papajons", "jons" → "Papajons" |

> El `ILIKE` es lo que permite encontrar "Papajons" al buscar "papa", "jons" o "papajo".  
> El índice trigrama (`gin_trgm_ops`) hace que ese `ILIKE` use el índice en vez de hacer full scan.

### Cambios en DB requeridos

Nueva migración `000XX_search_indexes.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Índice FTS en nombre de tiendas
CREATE INDEX idx_shops_name_fts
  ON business.shops USING GIN (to_tsvector('spanish', name));

-- Índice trigrama en nombre de tiendas (hace el ILIKE rápido)
CREATE INDEX idx_shops_name_trgm
  ON business.shops USING GIN (name gin_trgm_ops);

-- Índice FTS en nombre + descripción de productos
CREATE INDEX idx_items_search_fts
  ON products.items USING GIN (to_tsvector('spanish', name || ' ' || description));

-- Índice trigrama en nombre de productos
CREATE INDEX idx_items_name_trgm
  ON products.items USING GIN (name gin_trgm_ops);
```

---

## Diseño del endpoint

```
GET /shops/search?q=pizza&lat=10.0&lon=-67.0
```

**Auth:** Cookie de sesión, rol `Customer`.

**Query params:**

| Param | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `q` | string | sí | Texto a buscar |
| `lat` | float | sí | Latitud del usuario |
| `lon` | float | sí | Longitud del usuario |

**Respuesta:**
```json
{
  "shops": [
    {
      "id": "uuid",
      "name": "Papajons",
      "logo_image": "...",
      "address": "...",
      "score": 4.5,
      "distance": 1200,
      "duration": "8 min",
      "delivery_fee": 3.0,
      "matched_products": [
        { "id": "uuid", "name": "Pizza Margarita", "image": "...", "price": 15 }
      ]
    }
  ]
}
```

> `matched_products` se incluye solo cuando el match fue por producto, no por nombre de tienda.  
> `distance`, `duration` y `delivery_fee` siguen el mismo flujo que `calculateDistances` en `internal/shops/repo/shops.go`.

---

## Query SQL núcleo

```sql
SELECT DISTINCT ON (s.id)
  s.*,
  CASE
    WHEN (
      to_tsvector('spanish', s.name) @@ plainto_tsquery('spanish', $query)
      OR s.name ILIKE '%' || $query || '%'
    )
    THEN NULL
    ELSE json_agg(json_build_object(
      'id',    i.id,
      'name',  i.name,
      'image', i.image,
      'price', i.price
    ))
  END AS matched_products
FROM business.shops s
LEFT JOIN products.items i
  ON  i.shop_id      = s.id
  AND i.is_available = true
  AND i.deleted_at   IS NULL
WHERE s.opened         = true
  AND s.license_status = 'active'
  AND s.latitude       IS NOT NULL
  AND s.longitude      IS NOT NULL
  AND s.deleted_at     IS NULL
  -- Filtro geo 3 km (mismo Haversine del repo)
  AND 6371 * acos(
        cos(radians($lat)) * cos(radians(s.latitude)) *
        cos(radians(s.longitude) - radians($lon)) +
        sin(radians($lat)) * sin(radians(s.latitude))
      ) <= 3
  -- Match por nombre de tienda O nombre/descripción de producto
  AND (
    to_tsvector('spanish', s.name) @@ plainto_tsquery('spanish', $query)
    OR s.name ILIKE '%' || $query || '%'
    OR to_tsvector('spanish', i.name || ' ' || i.description) @@ plainto_tsquery('spanish', $query)
    OR i.name ILIKE '%' || $query || '%'
  )
GROUP BY s.id
ORDER BY s.id, s.score DESC
```

---

## Archivos a crear / modificar

### Nuevos

| Archivo | Propósito |
|---------|-----------|
| `internal/shops/domain/dto/search.go` | DTOs: `SearchQuery` y `SearchResponse` |
| `internal/shared/adapters/postgres/migrations/000XX_search_indexes.sql` | Extensión `pg_trgm` e índices GIN |

### Modificados

| Archivo | Cambio |
|---------|--------|
| `internal/shops/domain/ports/shops.go` | Agregar `SearchShops(ctx, query, coords)` a `ShopsRepo` y `ShopsApp` |
| `internal/shops/repo/shops.go` | Implementar `SearchShops` con la query + `calculateDistances` |
| `internal/shops/app/shops.go` | Delegar `SearchShops` al repo |
| `internal/shops/handler/shops.go` | Handler con bind/validate + Swagger comments |
| `api/router/groups/shops.go` | Registrar `GET /shops/search` |
| `postman/shops/shops.postman_collection.json` | Agregar request del nuevo endpoint |

> El mock de `ShopsRepo` se regenera con `mockery` tras tocar la interface.

---

## Caché

Mismo patrón Redis que el resto del repo:

- **Key:** `shops:search:<q_lowercase>:<lat_4dec>:<lon_4dec>`
- **TTL:** 2 minutos
- **Invalidación:** `cache.DeleteByPrefix("shops:search")` cuando una tienda cambia de estado o agrega un producto.

---

## Checklist de verificación

- [ ] `GET /shops/search?q=pizza&lat=X&lon=Y` retorna tiendas que venden pizza aunque no tengan "pizza" en el nombre
- [ ] `?q=papa` retorna "Papajons" (substring al inicio)
- [ ] `?q=jons` retorna "Papajons" (substring al final)
- [ ] `?q=papajons` retorna "Papajons" (nombre completo)
- [ ] Buscar el nombre de una tienda la retorna aunque no tenga productos con ese nombre
- [ ] Query sin resultados retorna `{ "shops": [] }` sin error
- [ ] Coordenadas fuera del radio de 3 km retornan lista vacía
- [ ] `go test ./internal/shops/...` pasa
