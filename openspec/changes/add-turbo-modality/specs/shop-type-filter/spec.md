## MODIFIED Requirements

### Requirement: Filtro opcional type en listado de tiendas
El sistema SHALL permitir filtrar el listado de tiendas cercanas por tipo mediante el query param opcional `type` en `GET /api/v1/shops/all`.

- **Método / path:** `GET /api/v1/shops/all`
- **Rol:** `Customer` (auth requerida)
- **Query:**
  - `lat` (number, required)
  - `lon` (number, required)
  - `type` (string, optional): `Restaurant` | `Supermarket` | `Turbo`
- **Response 200:** array de `ShopResponse` (incluye campo `type`)
- **Response 400:** coordenadas inválidas o `type` con valor no permitido
- **Radio:** 3 km para `Restaurant`/`Supermarket`/sin type; **2 km** cuando `type=Turbo`

#### Scenario: Solo restaurantes
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>&type=Restaurant`
- **THEN** la respuesta incluye únicamente tiendas con `type = "Restaurant"`

#### Scenario: Solo supermercados
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>&type=Supermarket`
- **THEN** la respuesta incluye únicamente tiendas con `type = "Supermarket"`

#### Scenario: Solo turbo con radio 2 km
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>&type=Turbo`
- **THEN** la respuesta incluye únicamente tiendas `type = "Turbo"` dentro de 2 km

#### Scenario: Sin type retorna Restaurant y Supermarket (no Turbo en feed general)
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>` sin query `type`
- **THEN** la respuesta incluye `Restaurant` y `Supermarket` a 3 km y **NO** incluye shops `Turbo` (Turbo se descubre vía `/turbo/entry`)

#### Scenario: type inválido
- **WHEN** un Customer envía `type` con un valor no permitido
- **THEN** el sistema responde HTTP 400

---

### Requirement: Filtro opcional type en búsqueda de tiendas
El sistema SHALL permitir filtrar la búsqueda de tiendas por tipo mediante el query param opcional `type` en `GET /api/v1/shops/search`.

- **Query `type` (optional):** `Restaurant` | `Supermarket` | `Turbo`
- Sin `type`: búsqueda en Restaurant + Supermarket (excluye Turbo del feed general), radio 3 km.
- Con `type=Turbo`: solo Turbo, radio 2 km.

#### Scenario: Búsqueda acotada a supermercado
- **WHEN** un Customer llama `GET /api/v1/shops/search?q=arroz&lat=<lat>&lon=<lon>&type=Supermarket`
- **THEN** solo se consideran tiendas con `type = "Supermarket"`

#### Scenario: Búsqueda turbo
- **WHEN** un Customer llama `GET /api/v1/shops/search?q=refresco&lat=<lat>&lon=<lon>&type=Turbo`
- **THEN** solo se consideran shops Turbo a ≤2 km

#### Scenario: type inválido en search
- **WHEN** un Customer envía `type=Pharmacy` en `/shops/search`
- **THEN** el sistema responde HTTP 400
