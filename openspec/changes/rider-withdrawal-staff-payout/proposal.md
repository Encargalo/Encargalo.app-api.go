## Why

Hoy `POST /riders/withdrawal` no es una solicitud: es una auto-liquidación. El rider la invoca y el sistema, en una sola transacción de BD, inserta un movimiento `withdrawal` con el saldo completo en negativo y deja el balance en 0 — sin que nadie apruebe, sin comprobante, sin dato de a quién pagarle, y sin ningún registro de que la plataforma le quedó debiendo ese dinero. El spec original (`rider-balance-commissions/specs/rider-withdrawal`) lo definió exactamente así y declaró el pago fuera de alcance ("solo se registra la solicitud"), por lo que la mitad operativa del ciclo nunca se construyó.

La consecuencia es que el dinero se evapora contablemente. La única lista del CRM es `ListPayables`, que filtra `WHERE entity_type = 'rider' AND amount > 0` (`internal/finance/repo/staff.go:112`), así que el rider desaparece de la lista en el instante en que solicita. Y aunque staff lo encuentre a mano por su `riderId`, `RecordPayout` valida con `money.EnsureDebitAllowed` contra un balance que ya está en 0 y rechaza el pago con `ErrInsufficientBalance` (`internal/finance/app/staff.go:81`). El retiro queda cerrado en el ledger antes de que exista la transferencia real. Ya hay al menos un caso así en Staging (rider `e3e4655b-5e9d-4b2e-ac21-50ad772c877a`, 2026-10-23 20:04 UTC).

El flujo operativo real es el mismo que ya funciona para los reembolsos de clientes: el rider solicita, la solicitud llega al CRM, staff hace la transferencia y carga el comprobante. Este cambio cierra ese ciclo reutilizando el molde de `finance.balance_refund_requests` en vez de inventar uno nuevo.

## What Changes

- **BREAKING** `POST /riders/withdrawal` deja de liquidar. Pasa a crear una solicitud en estado `pending` que retiene el saldo (movimiento `withdrawal_hold`) y queda registrada como deuda de la plataforma. La respuesta cambia de `{amount_withdrawn}` a la solicitud creada (id, monto, estado, datos de pago congelados). El nombre `amount_withdrawn` mentía: la plata nunca había salido.
- Nueva tabla `finance.withdrawal_requests`, genérica por `entity_id`/`entity_type` y espejo de `finance.balance_refund_requests`, con máquina de estados `pending → processing → completed` y la rama `pending|processing → rejected`. Un índice parcial único por entidad impide dos solicitudes activas a la vez — la regla anti-spam pasa a ser un constraint de BD, no lógica de aplicación sujeta a carreras.
- El ledger acompaña cada transición, igual que los reembolsos: `withdrawal_hold` (pending) → `withdrawal_processing` → `withdrawal_payout` (completed), o `withdrawal_restore` cuando se rechaza y el saldo vuelve al rider.
- **El cooldown de 24h se re-ancla**: hoy cuenta desde la última transacción `withdrawal`; pasa a contar desde `completed_at` de la última solicitud **pagada**. Una solicitud rechazada no quema el cooldown. Son dos reglas distintas: "una solicitud activa a la vez" (constraint) hace el trabajo anti-spam, y las 24h controlan el costo operativo de una transferencia manual por rider por día. Anclado en la solicitud, un pago rápido permitía dos transferencias en 12 horas; anclado en el pago, es exactamente una cada 24h.
- El monto sigue siendo el balance completo al momento de solicitar (retiro total, no parcial). Lo que el rider gane mientras su retiro está pendiente se acumula aparte para el siguiente.
- Nuevos endpoints de staff para el CRM: listar solicitudes por estado, completar con comprobante obligatorio (a Cloudinary, como `RecordPayout`), y rechazar con motivo.
- Los datos de pago se reutilizan de `users.accounts.pago_movil` vía `GET`/`PUT /users/pago-movil`, hoy limitado a `Require(models.Customer)` y que pasa a aceptar también `Rider`. Cada solicitud congela una copia (`pm_phone`, `pm_bank_code`, `pm_national_id`), así un cambio de teléfono posterior no reescribe la historia. Sin Pago Móvil registrado, la solicitud falla con `ErrPagoMovilRequired` — igual que los reembolsos.
- Se publica un evento al Redis Stream por cada solicitud nueva, para que `cmd/notifications` avise a Slack. Hoy un retiro no genera ninguna notificación.
- Migración con backfill: los movimientos `withdrawal` ya existentes se registran como solicitudes `completed` sin comprobante, para no dejar movimientos de ledger sin solicitud que los explique. Son datos de prueba —la plataforma todavía no opera— así que el backfill es automático y no requiere revisión caso por caso.

## Capabilities

### New Capabilities
- `withdrawal-requests`: ciclo de vida de una solicitud de retiro — estados, transiciones, invariantes (una activa por entidad), movimientos de ledger asociados y restauración de saldo al rechazar. Genérica por `entity_type` para que los comercios la hereden sin duplicarla.
- `withdrawal-staff-payout`: operación del CRM — listado de solicitudes por estado, pago con comprobante obligatorio y rechazo con motivo.
- `rider-payment-details`: el Pago Móvil del rider como destino del retiro, su captura y el congelado por solicitud.

### Modified Capabilities
(ninguna archivada — `openspec/specs/` está vacío. El comportamiento que este cambio reemplaza vive en `openspec/changes/rider-balance-commissions/specs/rider-withdrawal/spec.md`, que todavía no se archivó; las nuevas capabilities lo sustituyen y el `rider-withdrawal` viejo queda obsoleto al archivar este cambio.)

## Impact

- **Migración goose nueva** en `internal/shared/adapters/postgres/migrations/`: tabla `finance.withdrawal_requests`, índice parcial único de solicitud activa por `(entity_id, entity_type)`, índice `(status, created_at DESC)` para el listado del CRM, y backfill de los `withdrawal` existentes.
- `internal/riders/app/finance.go`: `RequestWithdrawal` deja de llamar a `RecordMovement` con `-balance` y pasa a crear la solicitud con hold. `GetLastWithdrawal` cambia de semántica (último **pagado**).
- `internal/riders/repo/finance.go`: `GetLastWithdrawal` consulta la tabla nueva por `status = 'completed'`.
- `internal/riders/domain/{models,dtos,ports}`: modelo de solicitud, estados, DTOs de request/response y puertos.
- `internal/finance/`: la tabla y sus movimientos viven acá (módulo genérico por `entity_type`); nuevos movement types en `domain/models/balance.go` y métodos de transición en `repo`/`app`.
- `internal/riders/handler/{finance.go,staff_finance.go}`: respuesta nueva de `RequestWithdrawal`; handlers de listado, complete y reject.
- `api/router/groups/riders.go`: rutas nuevas bajo `/riders/staff/finance/withdrawals`.
- `api/router/groups/users.go:67`: `GET`/`PUT /users/pago-movil` acepta `Rider` además de `Customer`.
- `internal/orders/repo/redis_stream.go` + `internal/notifications/`: evento y notificación de solicitud nueva.
- `internal/shared/errcustom/`: sentinels nuevos (solicitud activa existente, solicitud no encontrada, transición inválida) mapeados en `http.go`.
- Mocks de mockery regenerados; tests nuevos en `internal/test/` y `test/riders/handler/`. `internal/test/riders/app/finance_test.go:104` (`TestRequestWithdrawal_Success`) asserta el comportamiento viejo (`RecordMovement(..., -800.0, 600.0, ...)`) y debe reescribirse.
- Swagger regenerado + colección Postman de `riders` actualizada.
- **CRM** (`Encargalo.app-CRM`, repo aparte): consume los endpoints nuevos. La pantalla de payables deja de ser el único lugar donde mirar; las solicitudes de retiro necesitan su propia vista. Fuera del alcance de este change.
- El spec viejo pedía HTTP 201 y el handler devuelve 200 (`internal/riders/handler/finance.go:76`). Se fija el contrato explícitamente en el spec nuevo.
