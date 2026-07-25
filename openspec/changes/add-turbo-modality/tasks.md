## 1. Domain — shop types & radios

- [x] 1.1 Añadir `ShopTypeTurbo = "Turbo"` e incluirlo en `IsValidShopType`
- [x] 1.2 Constantes `CoverageRadiusKm = 3` y `TurboCoverageRadiusKm = 2`; usarlas en repo (dejar de hardcodear `3` donde aplique)
- [x] 1.3 Extender validación `oneof` de `ListShopsQuery` / `SearchShopsQuery` para incluir `Turbo`

## 2. Migration

- [x] 2.1 Migración goose: ampliar `shops_type_check` a `('Restaurant','Supermarket','Turbo')` (número siguiente al último existente)

## 3. Ports + App + Repo — assignment

- [x] 3.1 Añadir ports `ResolveTurboEntry(ctx, lat, lon) (TurboEntryResponse, error)` (o equivalente) en shops
- [x] 3.2 Repo: query shops `type=Turbo`, `opened`, `license active`, haversine ≤ 2 km, orden por distancia / score
- [x] 3.3 App: mapear `ShopResponse` con ETA **sin** +20; respuesta `{ available, shop, eta_minutes, coverage_km }`
- [x] 3.4 Listado/search: radio 2 km si `type=Turbo`; sin `type` excluir shops Turbo del feed general (solo Restaurant + Supermarket a 3 km)
- [x] 3.5 Regenerar mocks (`mockery`)

## 4. Orders — guards & ETA

- [x] 4.1 En quote/create: si shop es Turbo, validar distancia cliente–shop ≤ 2 km; error sentinel + mapeo HTTP
- [x] 4.2 Ajustar cálculo de `time` expuesto para Turbo (sin prep +20) en los paths que construyen `ShopResponse` / entry

## 5. Handler + Router

- [x] 5.1 Handler `GET /turbo/entry` (Customer) + Swagger
- [x] 5.2 Registrar grupo en `api/router` / DI si hace falta
- [x] 5.3 Actualizar Swagger de `/shops/all` y `/shops/search` (`type` incluye Turbo; nota de radios)

## 6. Tests

- [x] 6.1 Tests handler entry: disponible / no disponible / coords inválidas
- [x] 6.2 Tests app assignment: elige más cercano; ignora cerrados; respeta 2 km
- [x] 6.3 Tests orden: reject Turbo >2 km; accept ≤2 km
- [x] 6.4 `go test ./...` y `go build -v ./...`

## 7. Docs Frontend + Postman

- [x] 7.1 Crear `docs/frontend-turbo.md` (flujo Rappi: tile → entry → products → cart → pay)
- [x] 7.2 Postman: request `GET /turbo/entry`; actualizar shops `type=Turbo`
- [x] 7.3 `swag init -g cmd/server/main.go --parseDependency --parseInternal`
