## ADDED Requirements

### Requirement: Entrada Turbo asigna un nodo por geolocalización
El sistema SHALL exponer un endpoint de entrada Turbo que, dadas las coordenadas del Customer, asigne como máximo un shop `type = Turbo` abierto dentro de 2 km.

- **Método / path:** `GET /api/v1/turbo/entry`
- **Rol:** `Customer` (auth requerida)
- **Query:** `lat` (required), `lon` (required)
- **Response 200 (disponible):**
  ```json
  {
    "available": true,
    "shop": { "...ShopResponse": "incluye id, name, type=Turbo, time sin prep fija, ..." },
    "eta_minutes": 8,
    "coverage_km": 2
  }
  ```
- **Response 200 (no disponible):** `{ "available": false }`
- **Response 400:** coordenadas inválidas

#### Scenario: Asigna el Turbo más cercano abierto
- **WHEN** existen varios shops `type=Turbo` abiertos a ≤2 km del Customer
- **THEN** el sistema retorna `available=true` y el `shop` con menor distancia haversine (desempate por score / id)

#### Scenario: Fuera de cobertura
- **WHEN** no hay ningún Turbo abierto a ≤2 km
- **THEN** el sistema retorna `available=false` y no inventa un shop

#### Scenario: Turbo cerrado no se asigna
- **WHEN** el único Turbo ≤2 km tiene `opened=false`
- **THEN** el sistema retorna `available=false` (o el siguiente elegible si existe)

---

### Requirement: Radio Turbo en listado filtrado
Cuando el Customer lista o busca con `type=Turbo`, el sistema SHALL aplicar radio de **2 km** (no 3 km).

#### Scenario: Listado Turbo usa 2 km
- **WHEN** un Customer llama `GET /api/v1/shops/all?type=Turbo&lat=&lon=`
- **THEN** solo se incluyen shops Turbo dentro de 2 km
