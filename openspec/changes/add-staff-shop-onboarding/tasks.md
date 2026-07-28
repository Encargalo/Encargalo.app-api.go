## 1. Migration & Domain — Shop.Status

- [x] 1.1 Migración goose: agregar `status varchar default 'draft'` a `business.shops`; `UPDATE business.shops SET status = 'active'` (backfill de lo existente) en el `Up`; `Down` elimina la columna
- [x] 1.2 Añadir `Status` (`draft`/`active`) al modelo `Shop` (`internal/shops/domain/models/shops/shops.go`) y constantes `ShopStatusDraft`/`ShopStatusActive`
- [x] 1.3 Excluir shops `draft` de `GetAllShops`, `GetShopsBy`, `SearchShopCandidates` y `ListTurboCandidates` (`internal/shops/repo/shops.go`), igual que ya se filtra por `license_status`

## 2. Owner role assignment (users/auth)

- [x] 2.1 Puerto + método repo para buscar cuenta por teléfono devolviendo datos identificatorios (`StaffShopRepo.FindOwnerCandidateByPhone`, propio de shops en vez de extender `SearchAccountByPhone`, siguiendo el patrón ya usado por riders)
- [x] 2.2 Puerto + método repo para insertar una fila en `users.user_roles` (role Owner + shop_id) sobre una cuenta existente, sin tocar sus roles actuales (`StaffShopRepo.EnsureOwnerRole`, `ON CONFLICT (user_id, role_id, shop_id) DO UPDATE`)
- [x] 2.3 App layer: vinculación resuelta como parte de `StaffShopApp.CreateDraftShop` (busca por teléfono vía `FindOwnerCandidate`, luego crea+vincula en un solo flujo) en vez de un `LinkShopOwner` separado
- [x] 2.4 Mapear errores: teléfono sin cuenta asociada (404 `ErrNotFound`); `EnsureOwnerRole` es idempotente vía `ON CONFLICT ... DO UPDATE`

## 3. Shop staff onboarding (ports + app + repo)

- [x] 3.1 Puerto `StaffShopApp`/`StaffShopRepo` (dominio propio, separado de `ShopsApp`/`ShopsRepo` como hace riders con `RiderProfileApp`): `CreateDraftShop`, `UpdateDraftShop`, `ActivateShop`, `GetShop`, `ListShops`, `FindOwnerCandidate`
- [x] 3.2 App layer: `CreateDraftShop` crea con `status=draft` + vincula Owner; `UpdateDraftShop` actualiza solo los campos presentes; `ActivateShop` es idempotente si ya está `active` y exige `Completion().IsComplete`
- [x] 3.3 Repo: implementados sobre `business.shops` (`internal/shops/repo/staffShop.go`)
- [x] 3.4 Gate en `SetOpenClose` (`internal/shops/app/shops.go`): rechaza `open: true` si `shop.Status != active` vía nuevo `errcustom.ErrShopNotActive` (422)
- [x] 3.5 Regenerar mocks (`mockery`)

## 4. Staff shop impersonation (auth) — [REVERTIDO]

Todo lo emitido bajo este punto (sesión suplantada) se implementó y luego se eliminó por completo: el hand-off de cookie cross-origen entre CRM y Dashboard resultó inviable (ver Decisión #2 revisada en `design.md`). Se reemplaza por la sección 4bis.

- [x] 4.1 ~~Añadir `ActingStaffID`...~~ — eliminado de `ActiveSession`
- [x] 4.2 ~~`ImpersonateShop`...~~ — eliminado de `ports.AuthApp`/`appAuth`/handler
- [x] 4.3 ~~Middleware `BlockImpersonation`...~~ — eliminado de `AuthMiddleware`
- [x] 4.4 ~~Logging de `acting_staff_id`...~~ — eliminado del middleware `Auth`

## 4bis. Staff products management (products)

- [x] 4bis.1 Nuevo `StaffProductsHandler` (`internal/products/handler/staffProducts.go`) que reusa `ports.ProductsApp` tomando `shopID` de `c.Param("id")` en vez de `session.ShopID`: crear/listar/actualizar categorías, resumen de categorías, macro-categorías, productos del shop, crear/actualizar producto, crear/actualizar/borrar flavor, crear/actualizar/borrar descuento, catálogo de adiciones
- [x] 4bis.2 Update/Delete (categoría, item, flavor, descuento): se cambió la firma de los 6 métodos afectados en `app`/`ports` de `userID` a `shopID` explícito — se eliminó por completo la dependencia de `emplService.SearchEmployeeByID` en `productsApp` (ya no se inyecta). El handler de negocio (Owner/OrderManager) ahora pasa `*session.ShopID` directo (igual que ya hacían los `Register*`); el handler de staff pasa el `shopID` del path. Se agregó el check `session.ShopID == nil` a los 6 handlers de negocio afectados, consistente con el resto.
- [x] 4bis.3 DI: registrado `handProducts.NewStaffProductsHandler` en `cmd/server/providers/di.go`; `NewProductsGroup` recibe el nuevo handler
- [ ] 4bis.4 Logging: `staff_id` (`session.UserID`) + `shop_id` (path) + ruta y método en cada request de este grupo — pendiente, no bloqueante

## 5. Router

- [x] 5.1 Nuevo grupo `/shops/staff` (rol `Staff`): `GET /candidates`, `POST` crear draft (ya vincula owner), `GET` listar, `GET /:id`, `PATCH /:id` edición incremental, `POST /:id/activate`
- [x] 5.2 Nuevo grupo `/shops/staff/:id/products/*` (rol `Staff`) en `api/router/groups/products.go` — ver 4bis
- [x] 5.3 DI: registrados `handShops.NewStaffShopHandler`, `appShops.NewStaffShopApp`, `repoShops.NewStaffShopRepo`, `handProducts.NewStaffProductsHandler` en `cmd/server/providers/di.go`

## 6. Tests

- [x] 6.1 App: `CreateDraftShop`, `UpdateDraftShop` (parcial), `ActivateShop` (incompleto/completo/idempotente) — `internal/test/shops/app/staffShop_test.go`
- [x] 6.2 App: gate de `SetOpenClose` — rechaza en `draft`, permite en `active`, `close` no chequea status — `internal/test/shops/app/setOpenClose_gate_test.go`
- [x] 6.3 App: `Shop.Completion()` (vacío/parcial/completo) — `internal/test/shops/models/completion_test.go`
- [x] 6.4 ~~Handler: `ImpersonateShop`~~ — eliminado (`internal/test/auth/handler/impersonate_test.go` borrado)
- [x] 6.5 Handler: creación/activación de shop por staff (payload inválido, 422 con missing) — `internal/test/shops/handler/staffShop_test.go`
- [x] 6.5b Handler `products`: tests de `UpdateCategory`/`UpdateItem`/`UpdateFlavor`/`DeleteFlavor`/`UpdateItemDiscount`/`DeleteItemDiscount` migrados a `shopID` explícito — `internal/test/products/handler/products_test.go`
- [ ] 6.6 Handler: `StaffProductsHandler` — sin tests dedicados todavía (pendiente, no bloqueante; la lógica reusada por debajo ya está cubierta por los tests del handler de negocio)
- [x] 6.7 `go test ./...` y `go build -v ./...` — verde

## 7. Docs

- [x] 7.1 Swagger godoc en todos los handlers nuevos, incluyendo `StaffProductsHandler`
- [x] 7.2 Postman: nuevas requests `[Staff] *` en `postman/products/products.postman_collection.json`
- [x] 7.3 `swag init -g cmd/server/main.go --parseDependency --parseInternal`
