## 1. Domain — shops

- [x] 1.1 Crear `internal/shops/domain/models/shops/shopType.go` con constantes `Restaurant` / `Supermarket` e `IsValidShopType`
- [x] 1.2 Añadir DTO `ListShopsQuery` (o equivalente) con `lat`, `lon` y `type` opcional (`omitempty,oneof=Restaurant Supermarket`) sin contaminar `Coords` usado en `GetShopsBy`
- [x] 1.3 Extender `SearchShopsQuery` con `type` opcional y la misma validación `oneof`

## 2. Ports

- [x] 2.1 Actualizar firmas de `ShopsApp.GetAllShops` / `ShopsRepo.GetAllShops` para aceptar el query con `type` (o pasar `type` además de coords)
- [x] 2.2 Confirmar que `SearchShops` ya recibe `SearchShopsQuery` (solo propagar el nuevo campo; ajustar ports si hace falta)

## 3. Migration

- [x] 3.1 Crear migración goose `00051_shop_type_supermarket_and_macros.sql`: drop/recreate `shops_type_check` para `('Restaurant','Supermarket')`
- [x] 3.2 En la misma migración (o `00052_...`), seed de macros supermercado: `abarrotes`, `frescos`, `lacteos`, `bebidas-super`, `aseo`, `otros-super` (IDs fijos, down reversible)

## 4. App + Repo

- [x] 4.1 En `shops` repo: aplicar `WHERE type = ?` en `GetAllShops` y `SearchShopCandidates` solo si `type` no está vacío
- [x] 4.2 En `shops` app: validar/propagar `type` desde handlers hacia repo
- [x] 4.3 Regenerar mocks con `mockery` tras cambios de ports

## 5. Handler + Router

- [x] 5.1 Actualizar `GetAllShops` handler para bind/validate del query con `type` y Swagger (`@Param type`)
- [x] 5.2 Actualizar `SearchShops` handler Swagger/descripción para `type` opcional
- [x] 5.3 Verificar que el router no requiere cambios de path (solo query)

## 6. Tests

- [x] 6.1 Tests de handler: `type` válido filtra; omitido OK; inválido → 400 (`GetAllShops` y `SearchShops`)
- [x] 6.2 Tests de app/repo (o unit del filtro) según patrón del módulo shops existente
- [x] 6.3 `go test ./...`
- [x] 6.4 `go build -v ./...`

## 7. Docs

- [x] 7.1 Actualizar Postman `postman/shops/` (o colección shops) con `type` en `/shops/all` y `/shops/search`; `base_url` = `http://localhost:3000`
- [x] 7.2 Regenerar Swagger: `swag init -g cmd/server/main.go --parseDependency --parseInternal`
