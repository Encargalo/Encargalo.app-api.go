## 1. Esquema y migración

- [x] 1.1 Crear migración goose en `internal/shared/adapters/postgres/migrations/` con la tabla `finance.withdrawal_requests` (`id`, `entity_id`, `entity_type`, `amount`, `currency`, `bcv_rate`, `status` default `pending`, `pm_phone`, `pm_bank_code`, `pm_national_id`, `proof_url`, `staff_id`, `rejection_reason`, `created_at`, `processing_at`, `completed_at`, `resolved_at`), espejo de `finance.balance_refund_requests`; verificar con `goose up` seguido de `goose down` sobre una BD local, que ambos corran sin error
- [x] 1.2 Agregar en la misma migración el índice parcial único `(entity_id, entity_type) WHERE status IN ('pending','processing')` y el índice `(status, created_at DESC)`; verificar con dos `INSERT` manuales de solicitudes `pending` para la misma entidad, que el segundo falle por violación de unicidad
- [x] 1.3 Agregar el índice parcial único de reserva sobre `finance.transactions (reference_id) WHERE reference_type = 'withdrawal_request' AND movement_type IN ('withdrawal_hold','withdrawal_processing','withdrawal_payout')`, espejo de `idx_finance_tx_refund_reservation_unique`; verificar que un segundo movimiento de reserva para la misma solicitud sea rechazado por la BD
- [x] 1.4 Backfill de los movimientos `withdrawal` existentes a solicitudes `completed` sin comprobante, con el movimiento reetiquetado a `withdrawal_payout` y su `reference_id`. Son datos de prueba (la plataforma todavía no opera), así que no requiere revisión caso por caso; verificar sobre la BD de Staging que cada movimiento `withdrawal` quede con exactamente una solicitud asociada y que los balances no cambien

## 2. Dominio y modelos

- [x] 2.1 Crear el modelo `WithdrawalRequest` y el tipo `WithdrawalStatus` (`pending`, `processing`, `completed`, `rejected`) en `internal/finance/domain/models/`, con el struct bun apuntando a `finance.withdrawal_requests`; verificar que `go build ./...` compile
- [x] 2.2 Agregar los movement types `withdrawal_hold`, `withdrawal_processing`, `withdrawal_payout` y `withdrawal_restore` en `internal/finance/domain/models/balance.go` y la constante de `reference_type` `withdrawal_request`; verificar que compile
- [x] 2.3 Implementar la función de validación de transiciones de estado (permitidas: `pending→processing`, `processing→completed`, `pending→rejected`, `processing→rejected`; terminales: `completed`, `rejected`; `pending→completed` prohibido) como método del dominio; verificar con tests unitarios que cubran cada transición permitida y al menos tres prohibidas, incluyendo el pago sin reclamo previo
- [x] 2.4 Agregar `reference_id`/`reference_type` al struct `Transaction` de `internal/riders/domain/models/balance.go` para que el historial del rider pueda exponer el vínculo con la solicitud; verificar con un test de lectura que los campos se hidraten
- [x] 2.5 Crear los DTOs en `internal/finance/domain/dtos/` y `internal/riders/domain/dtos/`: respuesta de solicitud creada, ítem de listado para staff, request de complete (comprobante) y de reject (motivo), y el bloque de solicitud activa que `GET /riders/balance` agrega a `BalanceResponse`; verificar que compile
- [x] 2.6 Agregar los sentinels `ErrWithdrawalAlreadyRequested` (409), `ErrWithdrawalNotFound` (404), `ErrWithdrawalInvalidTransition` (409), `ErrWithdrawalProofRequired` (400) y `ErrWithdrawalRejectionReasonRequired` (400) en `internal/shared/errcustom/errors.go` con su mapeo de status y código en `http.go`; verificar con un test de tabla que cada sentinel resuelva al status y código esperados
- [x] 2.7 Declarar los puertos en `internal/finance/domain/ports/` (repo: crear solicitud con hold, obtener por id, listar por estado y entity_type, transicionar a processing/completed/rejected, último `completed` por entidad; app: las operaciones de staff) y en `internal/riders/domain/ports/` el puerto que `riders/app` consume; verificar que compile

## 3. Capa de persistencia

- [x] 3.1 Implementar en `internal/finance/repo/` la creación de la solicitud con hold: dentro de una sola transacción, `SELECT ... FOR UPDATE` del balance, validación del mínimo, inserción de la solicitud `pending`, movimiento `withdrawal_hold` con `reference_id`/`reference_type`, y actualización del balance; verificar con un test de integración que balance y solicitud queden consistentes y que el balance se lea dentro de la transacción (no antes)
- [x] 3.2 Implementar la transición a `completed`: `SELECT ... FOR UPDATE` de la solicitud, validación de estado, escritura de `status`/`staff_id`/`proof_url`/`completed_at` y movimiento `withdrawal_payout`; verificar con un test que una segunda llamada concurrente falle con `ErrWithdrawalInvalidTransition` y que el índice de reserva impida un segundo movimiento de pago
- [x] 3.3 Implementar la transición a `rejected` con devolución de saldo: escritura de `status`/`staff_id`/`rejection_reason`/`resolved_at` y movimiento `withdrawal_restore` positivo; verificar con un test que el balance vuelva al valor previo más los movimientos intermedios
- [x] 3.4 Implementar el reclamo (`pending → processing`) que calcula el monto pagable neto como `retenido + min(disponible, 0)` bajo `SELECT ... FOR UPDATE` y lo congela en la fila; verificar con tests que un efectivo cobrado antes del reclamo reduzca el monto, que ganancias bancarias posteriores no lo aumenten, y que un segundo reclamo falle con `ErrWithdrawalInvalidTransition`
- [x] 3.5 Implementar la anulación automática cuando el monto pagable neto resulta cero o negativo al reclamar: la solicitud no pasa a `processing`, el saldo negativo queda en el balance del rider y el cooldown no se activa; verificar con un test que el rider pueda volver a solicitar sin esperar 24 horas
- [x] 3.6 Garantizar que el pago registre el monto congelado al reclamar sin recalcular; verificar con un test que debita el balance del rider entre el reclamo y el pago, que el movimiento `withdrawal_payout` sea por el monto congelado y el balance quede negativo por la diferencia
- [x] 3.7 Implementar el listado por estado y `entity_type` con join a `users.accounts` para nombre, apellido y teléfono, ordenado por `created_at DESC`, exponiendo monto solicitado y monto pagable por separado; verificar con un test que el filtro por estado funcione, que el listado sin filtro devuelva solo `pending` y `processing`, y que el pagable de una solicitud ya reclamada sea el congelado
- [x] 3.8 Reemplazar `GetLastWithdrawal` en `internal/riders/repo/finance.go` por la consulta de la última solicitud `completed` ordenada por `completed_at`; verificar con un test que una solicitud `rejected` reciente no active el cooldown y que una `completed` reciente sí
- [x] 3.9 Implementar la consulta de solicitud activa por entidad (para el bloque de `GET /riders/balance` y para la validación de solicitud única); verificar con un test que devuelva la `pending` o `processing` existente y nada cuando no hay

## 4. Lógica de negocio

- [x] 4.1 Reescribir `RequestWithdrawal` en `internal/riders/app/finance.go`: validar cooldown contra el último pago, obtener el Pago Móvil de la cuenta y fallar con `ErrPagoMovilRequired` si no existe, resolver la tasa BCV y el mínimo, y delegar la creación con hold al repo congelando los datos de pago. Eliminar el `RecordMovement(-balance, minBs)` actual; verificar con tests unitarios los escenarios del spec `withdrawal-requests` (éxito, mínimo, negativo, solicitud activa, cooldown, sin Pago Móvil, tasa no disponible)
- [x] 4.2 Reescribir `internal/test/riders/app/finance_test.go:104` (`TestRequestWithdrawal_Success`), que asserta el comportamiento viejo `RecordMovement(..., -800.0, 600.0, ...)`; verificar que el test nuevo falle contra la implementación vieja y pase contra la nueva
- [x] 4.3 Implementar en `internal/finance/app/` las operaciones de staff: listar, completar (subida del comprobante a Cloudinary vía el uploader existente, validación con `uploadimage.ValidateCloudinaryURL`, comprobante obligatorio) y rechazar (motivo obligatorio); verificar con tests unitarios que un complete sin comprobante y un reject sin motivo fallen antes de tocar la BD, y que una falla de subida deje la solicitud intacta
- [x] 4.4 Agregar el bloque de solicitud activa a la respuesta de `GetBalance` en `internal/riders/app/finance.go`; verificar con un test que un rider con retiro `pending` vea monto y estado, y uno sin solicitudes vea el campo vacío
- [x] 4.5 Publicar el evento de solicitud nueva al Redis Stream desde `RequestWithdrawal`, sin que una falla de publicación aborte la solicitud; verificar con un test que el retiro se cree aunque el productor devuelva error

## 5. Capa HTTP

- [x] 5.1 Actualizar `RequestWithdrawal` en `internal/riders/handler/finance.go` para devolver HTTP 201 con la solicitud creada en vez de 200 con `{amount_withdrawn}`, con anotaciones Swagger completas (`@Success 201`, `@Failure 409/422/429/503`); verificar con un test de handler que el status y el cuerpo sean los nuevos
- [x] 5.2 Implementar en `internal/riders/handler/staff_finance.go` el handler de listado (`status` como query param) con Swagger; verificar con un test de handler usando `buildhttptest.SetupControllerCase` que el filtro llegue al app layer
- [x] 5.3 Implementar los handlers de claim, complete y reject con Swagger, mapeando los sentinels nuevos; verificar con tests de handler los casos 400 (sin comprobante / sin motivo), 404 (inexistente) y 409 (reclamo doble y pago sin reclamo previo)
- [x] 5.4 Registrar las rutas en `api/router/groups/riders.go` bajo `/riders/staff/finance/withdrawals` (listado, `:id/claim`, `:id/complete`, `:id/reject`) con `Auth + Require(models.Staff)`; verificar levantando el servidor que las rutas respondan 401 sin sesión de staff
- [x] 5.5 Habilitar el rol `Rider` en `GET`/`PUT /users/pago-movil` en `api/router/groups/users.go:67` sin perder el acceso de `Customer`; verificar con una llamada autenticada como rider que devuelva 200/404 y no 401
- [x] 5.6 Registrar las dependencias nuevas en `cmd/server/providers/di.go`; verificar que el servidor arranque sin error de resolución del contenedor

## 6. Notificaciones

- [x] 6.1 Mapear el evento de solicitud de retiro en `internal/notifications/handler` y su `models.NotificationMessage` en `app`, con rider, monto y identificador de solicitud; verificar con un test unitario del mapeo
- [x] 6.2 Verificar el camino completo publicando un evento de prueba en el stream y comprobando que `cmd/notifications` lo consuma y envíe el mensaje a Slack en dev

## 7. Mocks, suite y documentación

- [x] 7.1 Regenerar mocks con `mockery` para los puertos nuevos y modificados; verificar que `go build ./...` y `go test ./...` compilen
- [x] 7.2 Correr la suite completa (`go test ./...`) y dejarla verde, incluyendo los tests de finance y riders existentes que tocan balances
- [x] 7.3 Regenerar Swagger con `go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/server/main.go --parseDependency --parseInternal`; verificar que los endpoints nuevos y el 201 de `POST /riders/withdrawal` aparezcan en `docs/`
- [ ] 7.4 Actualizar la colección Postman del dominio `riders` en el repo `Encargalo.app-api.postman` (`riders/riders.postman_collection.json`) con los tres endpoints nuevos y el cuerpo actualizado de `POST /riders/withdrawal`, usando `{{base_url}}`; verificar importando la colección que las requests resuelvan contra el entorno Local
- [x] 7.5 Actualizar la sección de retiros en `CLAUDE.md` describiendo el ciclo nuevo y la relación con el flujo de reembolsos, para que el split de responsabilidades quede documentado como el de trips/riders

## 8. Cierre de la deuda en dev

- [ ] 8.1 Resolver manualmente el retiro huérfano de Staging (rider `e3e4655b-5e9d-4b2e-ac21-50ad772c877a`, movimiento del 2026-10-23 20:04 UTC) una vez aplicada la migración; verificar que aparezca en el listado del CRM con el estado que corresponda
- [ ] 8.2 Validar el ciclo completo en Staging de punta a punta: registrar Pago Móvil como rider, solicitar, ver la solicitud en el listado de staff, completar con comprobante y comprobar que el cooldown de 24h arranque desde el pago; verificar también el camino de rechazo y que el saldo vuelva
