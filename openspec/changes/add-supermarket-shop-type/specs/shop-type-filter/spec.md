## ADDED Requirements

### Requirement: Filtro opcional type en listado de tiendas
El sistema SHALL permitir filtrar el listado de tiendas cercanas por tipo mediante el query param opcional `type` en `GET /api/v1/shops/all`.

- **Método / path:** `GET /api/v1/shops/all`
- **Rol:** `Customer` (auth requerida)
- **Query:**
  - `lat` (number, required)
  - `lon` (number, required)
  - `type` (string, optional): `Restaurant` | `Supermarket`
- **Response 200:** array de `ShopResponse` (incluye campo `type`)
- **Response 400:** coordenadas inválidas o `type` con valor no permitido

#### Scenario: Solo restaurantes
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>&type=Restaurant`
- **THEN** la respuesta incluye únicamente tiendas con `type = "Restaurant"`

#### Scenario: Solo supermercados
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>&type=Supermarket`
- **THEN** la respuesta incluye únicamente tiendas con `type = "Supermarket"`

#### Scenario: Sin type retorna ambos
- **WHEN** un Customer llama `GET /api/v1/shops/all?lat=<lat>&lon=<lon>` sin query `type`
- **THEN** la respuesta puede incluir tiendas `Restaurant` y `Supermarket`

#### Scenario: type inválido
- **WHEN** un Customer envía `type` con un valor distinto de `Restaurant` o `Supermarket`
- **THEN** el sistema responde HTTP 400

---

### Requirement: Filtro opcional type en búsqueda de tiendas
El sistema SHALL permitir filtrar la búsqueda de tiendas por tipo mediante el query param opcional `type` en `GET /api/v1/shops/search`.

- **Método / path:** `GET /api/v1/shops/search`
- **Rol:** `Customer` (auth requerida)
- **Query:**
  - `q` (string, required)
  - `lat` (number, required)
  - `lon` (number, required)
  - `type` (string, optional): `Restaurant` | `Supermarket`
- **Response 200:** array de `SearchShopResponse`
- **Response 400:** query inválida o `type` no permitido

#### Scenario: Búsqueda acotada a supermercado
- **WHEN** un Customer llama `GET /api/v1/shops/search?q=arroz&lat=<lat>&lon=<lon>&type=Supermarket`
- **THEN** solo se consideran tiendas con `type = "Supermarket"` (y sus productos coincidentes)

#### Scenario: Búsqueda sin type incluye ambos tipos
- **WHEN** un Customer llama `GET /api/v1/shops/search?q=arroz&lat=<lat>&lon=<lon>` sin `type`
- **THEN** la búsqueda considera tiendas `Restaurant` y `Supermarket`

#### Scenario: type inválido en search
- **WHEN** un Customer envía `type=Pharmacy` en `/shops/search`
- **THEN** el sistema responde HTTP 400
