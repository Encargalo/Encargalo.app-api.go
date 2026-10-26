## Context

Ver `proposal.md` — Why para la motivación. Lo que condiciona el diseño:

- El esquema `finance` ya es genérico por `entity_type` (`rider`, `shop`, `customer`) y lo comparten `internal/riders`, `internal/shops`, `internal/users` e `internal/finance`. Cada módulo tiene su propio struct bun apuntando a las mismas tablas `finance.balances` y `finance.transactions`, con los campos que ese módulo necesita.
- El flujo de reembolsos de clientes ya resolvió exactamente este problema: `finance.balance_refund_requests` (migración `00046`) con máquina de estados, índice parcial único de solicitud activa, datos de Pago Móvil congelados, comprobante, staff responsable y motivo de rechazo; más cuatro movement types que espejan los estados (`internal/users/domain/models/finance.go:18-21`). Hay precedente funcionando, con tests, en producción.
- `finance.transactions` ya tiene `reference_id`/`reference_type` (migración `00075`), introducidos justamente para generalizar `order_id` a cualquier referencia, y `staff_id`/`notes`/`proof_url` (migración `00064`).
- `RecordMovement` (en `internal/finance/repo/staff.go` y su gemelo en `internal/riders/repo/finance.go`) hace `SELECT ... FOR UPDATE` sobre el balance, valida un mínimo opcional y escribe balance + movimiento en una sola transacción. Es la primitiva sobre la que se construye todo lo demás.
- `openspec/specs/` está vacío: no hay capabilities archivadas, así que este cambio no edita deltas de specs existentes.

## Goals / Non-Goals

**Goals:**

- Que una solicitud de retiro sea un registro con estado propio, consultable, y que la deuda de la plataforma con el rider sea visible en todo momento.
- Que el saldo nunca pueda retenerse ni liberarse dos veces por el mismo retiro, garantizado por el almacenamiento y no por lógica de aplicación.
- Que la máquina de estados y la tabla sirvan a comercios sin reescribirla, dado que `/shops/staff/finance/payables` tiene el mismo problema latente.
- Que el ciclo nuevo sea indistinguible del de reembolsos para quien ya conoce ese código.

**Non-Goals:**

- Retiro parcial. El monto sigue siendo el balance completo (decisión tomada con el usuario).
- Pago automatizado contra una API bancaria. La transferencia la sigue haciendo una persona; acá solo se registra.
- Pantallas del CRM. Viven en `Encargalo.app-CRM` y consumen estos endpoints.
- Migrar el flujo de retiro de comercios. La tabla queda lista para ello, pero exponerlo es otro cambio.
- WebSocket para staff. Los reembolsos lo usan porque tienen ventana de 15 minutos; un retiro no es time-critical y un listado REST + aviso a Slack alcanza.

## Decisions

### Decisión 1: tabla propia con columna `status`, no el `movement_type` como estado

`finance.withdrawal_requests` con una columna `status` real, en lugar de inferir el estado del `movement_type` del movimiento más reciente.

**Por qué:** el CRM necesita "dame todas las solicitudes pendientes" como una consulta indexada por estado, no un escaneo de `finance.transactions` adivinando por tipo de movimiento. Además una solicitud carga datos que no son del ledger: los datos de Pago Móvil congelados, el motivo de rechazo, las fechas de cada transición y el staff responsable. Es exactamente el razonamiento que llevó a `balance_refund_requests`, y replicarlo mantiene un solo patrón mental para los dos flujos.

**Alternativa descartada:** solo movement types sobre `finance.transactions`, mutando el tipo en cada transición como hace la migración `00050` para los reembolsos. Esa migración existe porque el flujo de reembolsos *ya tenía* una tabla de solicitudes y los tipos acompañan al estado — no lo reemplazan. Sin tabla, el estado quedaría implícito en la última fila escrita y cualquier consulta del CRM sería un escaneo con subconsulta.

### Decisión 2: genérica por `entity_id`/`entity_type` desde el día uno

La tabla no es `rider_withdrawal_requests`. Lleva `entity_id` + `entity_type`, igual que `finance.balances` y `finance.transactions`.

**Por qué:** los comercios ya tienen payables y el mismo agujero (su saldo también se paga a mano vía `RecordPayout`). El costo de generalizar ahora son dos columnas en lugar de una; el costo de no hacerlo es una segunda tabla, una segunda máquina de estados y una segunda batería de tests dentro de unos meses. La lógica vive en `internal/finance` (módulo genérico) y `internal/riders` la consume para `entity_type = 'rider'`.

**Alternativa descartada:** tabla específica de riders en `internal/riders`. Más corta de escribir hoy, duplicada mañana. Y `balance_refund_requests` ya demostró el costo de amarrar la tabla a un actor: nació con `customer_id` y no sirve para nadie más.

### Decisión 3: el saldo se retiene al solicitar (hold), no al pagar

Al crearse la solicitud se registra `withdrawal_hold` con el monto en negativo y el balance disponible baja. La deuda no desaparece: queda en la solicitud `pending`.

**Por qué:** si el balance no se tocara hasta el pago, el monto que staff ve y el que el rider tiene dejarían de coincidir en cuanto el rider complete un viaje más, y habría que decidir —en el momento de pagar— si se paga el monto solicitado o el saldo actual. Con el hold, lo solicitado queda inmóvil y lo que el rider gane después se acumula limpio para el siguiente retiro. Es el mismo razonamiento de `balance_refund_hold`.

Esto se parece al comportamiento actual (balance a 0 al solicitar) pero no es lo mismo: hoy el dinero se va sin dejar registro de la deuda. Con el hold hay una fila `pending` que dice exactamente cuánto se debe, a quién y desde cuándo, el rider lo ve en su balance, y un rechazo lo devuelve.

**Alternativa descartada:** no tocar el balance hasta el pago. Deja el saldo "comprometido" invisible y permite que el rider siga viendo como disponible un dinero que ya pidió.

### Decisión 4: el cooldown se ancla en `completed_at`, no en `created_at`

`GetLastWithdrawal` cambia de "última transacción `withdrawal`" a "última solicitud `completed`, por `completed_at`".

**Por qué:** las 24 horas controlan el costo operativo de una transferencia manual, y ese costo se incurre al pagar. Anclado en la solicitud, si staff paga rápido el rider puede volver a pedir pocas horas después del pago y terminan siendo dos transferencias en el mismo día — justo lo que la regla quería evitar. Anclado en el pago es exactamente una cada 24h. Y el trabajo anti-spam que antes hacía el cooldown ahora lo hace, mejor, el constraint de solicitud activa única.

Un rechazo no cuenta: si staff rechaza por datos mal cargados, bloquear al rider 24 horas lo castiga por un error ajeno.

### Decisión 5: `processing` es obligatorio y congela el monto

No existe `pending → completed` directo. Staff SIEMPRE reclama la solicitud primero (`pending → processing`), y ese reclamo escribe en la fila el monto pagable calculado en ese instante. El pago posterior registra ese monto congelado sin recalcular.

**Por qué:** la transferencia bancaria ocurre fuera del sistema. El operador mira la pantalla, se va a su app del banco, transfiere y vuelve a marcar el pago — minutos después. Si en esa ventana el rider completa un pedido en efectivo, el monto neto cambia, y recalcular al registrar el pago haría que los libros digan 72 Bs cuando del banco salieron 800. El descuadre lo crearía el propio sistema. Congelar al reclamar hace vinculante el número que el operador efectivamente vio y transfirió.

Como efecto secundario, el reclamo también evita que dos personas del equipo paguen la misma solicitud, que era la razón original por la que el estado existía.

**Alternativa descartada:** `processing` opcional, con el monto neto recalculado al pagar. Era la versión inicial de este diseño y es incorrecta por lo anterior: solo funciona si el pago fuera atómico con el registro, y no lo es.

**Alternativa descartada:** bloquear carreras en efectivo mientras hay una solicitud en `processing`. Elimina la carrera por completo y la ventana son minutos, pero mete el estado del retiro dentro de la lógica de asignación, que vive en otro dominio. El escenario además exige que el rider *complete* un pedido en efectivo dentro de esos minutos. Se puede agregar después sin rehacer nada si la operación lo pide.

### Decisión 5b: el monto solicitado es un techo, no una promesa

El monto congelado al crear la solicitud es lo que la plataforma debía en ese momento. Al reclamar, el monto pagable se recalcula como `retenido + min(disponible, 0)`.

**Por qué:** un pedido en efectivo debita el balance y lo hace fuerte — el rider cobra el total al cliente y queda debiendo todo lo que no es su pago (`CashOwedUSD`), así que un pedido de $20 con domicilio de $2 le debita $18.20. Si el rider pide retiro y después toma trabajo en efectivo, pagarle el monto solicitado completo significaría transferirle 800 Bs a alguien que en ese mismo momento tiene 728 Bs de la empresa en el bolsillo. La fórmula paga la posición neta real y salda la deuda contra el propio retiro.

Si el neto da cero o negativo (el efectivo se comió el retiro entero), no hay nada que transferir: la solicitud se anula automáticamente, el rider queda con su deuda y el cooldown NO se activa, porque no se le pagó nada. Con el piso en $3 este caso será común — un retiro de 120 Bs lo borra un solo pedido en efectivo.

### Decisión 6: el vínculo ledger ↔ solicitud va por `reference_id`/`reference_type`

Los cuatro movimientos del ciclo se escriben con `reference_type = 'withdrawal_request'` y `reference_id = <id de la solicitud>`. No se agrega una columna `withdrawal_request_id`.

**Por qué:** esas columnas se introdujeron en `00075` precisamente para generalizar este tipo de vínculo, y usarlas evita que `finance.transactions` siga acumulando una FK por cada flujo nuevo (ya tiene `order_id`, `trip_id` y `refund_request_id`). La invariante de "una sola reserva por solicitud" se escribe como índice parcial único sobre `reference_id`, espejo de `idx_finance_tx_refund_reservation_unique`.

**Consecuencia a atender:** el struct bun de `internal/riders/domain/models/balance.go` no tiene `reference_id`/`reference_type` (tiene `order_id` y `trip_id`). Hay que agregárselos para que `GET /riders/transactions` pueda exponer el vínculo. Los movimientos los escribe el repo de `internal/finance`, que ya los tiene.

### Decisión 7: `users.accounts.pago_movil` como destino, congelado por solicitud

Se reutiliza el campo y los endpoints existentes, habilitando el rol `Rider` en `api/router/groups/users.go:67`.

**Por qué:** un rider es una cuenta en `users.accounts`; el campo, el DTO, la validación y el sentinel `ErrPagoMovilRequired` ya existen y los reembolsos ya los ejercitan. Agregar campos bancarios a `riders.profiles` sería un segundo lugar donde vive el mismo dato, con dos formas de estar desactualizado.

**Alternativa descartada:** capturar los datos bancarios en el onboarding de rider (`riders.profiles`). Tiene sentido pedirlos al postularse, pero eso es una mejora de onboarding independiente; nada impide hacerla después escribiendo en el mismo campo de la cuenta.

### Decisión 8: el aviso sale por el Redis Stream existente

Se publica un evento al crear la solicitud y `cmd/notifications` lo mapea a un mensaje de Slack, como ya hace con pedidos.

**Por qué:** la tubería existe completa (productor, stream, consumer, fanout) y el consumer ya consulta OLTP directamente para armar sus mensajes. Una falla al publicar no debe abortar la solicitud: el retiro ya está registrado y el CRM lo muestra igual; la notificación es conveniencia, no fuente de verdad.

## Risks / Trade-offs

- **Cambio incompatible en `POST /riders/withdrawal`** (la respuesta deja de ser `{amount_withdrawn}` y el código pasa a 201) → está marcado **BREAKING** en el proposal, pero el riesgo real es bajo: la plataforma todavía no opera, así que no hay riders en producción usando el endpoint. Igual la app de riders debe actualizarse antes del lanzamiento.

- **Backfill de retiros existentes**: los movimientos `withdrawal` que existen hoy son datos de prueba, no deudas reales — la plataforma no está operando. El backfill puede marcarlos `completed` sin comprobante de forma automática, sin revisión caso por caso. Marcarlos `pending` sería peor: llenaría el CRM de solicitudes de prueba y dispararía avisos. Este riesgo desaparece si el cambio se implementa antes del lanzamiento, y se vuelve real si se posterga.

- **Dos riders, un mismo staff, pagos simultáneos**: el índice parcial único protege contra dos solicitudes activas por rider, pero no contra que dos personas del equipo paguen la misma solicitud a la vez → la transición de estado se hace bajo `SELECT ... FOR UPDATE` de la fila de la solicitud, y el índice único de reserva sobre `reference_id` impide un segundo movimiento de pago aunque la validación de estado se escape. El reclamo obligatorio (Decisión 5) cierra la ventana antes de que alguien llegue siquiera al banco.

- **La transferencia ocurre fuera del sistema**: entre que el operador ve el monto y marca el pago pasan minutos en los que el balance del rider puede moverse. No hay forma de hacerlo atómico → se mitiga congelando el monto al reclamar, no eliminando la ventana. Lo que caiga dentro de ella (típicamente un pedido en efectivo completado) se registra como deuda del rider en su balance, visible y saldable contra ganancias futuras, en vez de absorberse en silencio. Si el operador transfiere un monto distinto al congelado, la corrección va por `RecordAdjustment`, que ya existe, con el comprobante bancario como evidencia.

- **Solicitud reclamada y abandonada**: una solicitud en `processing` cuyo operador nunca vuelve queda congelada y bloquea al rider (cuenta como solicitud activa) → conviene un aviso cuando una solicitud lleva demasiado tiempo en `processing`, y que el rechazo siga disponible desde ese estado para liberarla devolviendo el saldo.

- **Saldo negativo heredado**: la implementación actual calcula el delta con un balance leído *fuera* de la transacción (`internal/riders/app/finance.go:215`), así que un movimiento intermedio puede dejar el balance negativo. El rediseño lee el balance dentro de la misma transacción que crea la solicitud, lo que cierra esa carrera de paso.

- **El rider pierde visibilidad si el CRM no se actualiza**: los endpoints nuevos no sirven de nada si nadie los consume. El aviso a Slack mitiga el caso "nadie estaba mirando", pero la vista del CRM es prerequisito operativo real para desplegar esto a producción.

## Migration Plan

1. Migración goose que crea `finance.withdrawal_requests`, sus índices (único parcial de solicitud activa, `(status, created_at DESC)` para el CRM) y el índice único de reserva sobre `finance.transactions`.
2. Backfill en la misma migración: una solicitud `completed` por cada movimiento `withdrawal` existente, con `completed_at = created_at` del movimiento, sin comprobante y sin `staff_id`, y el movimiento reetiquetado a `withdrawal_payout` con su `reference_id` apuntando a la solicitud creada. Automático: son datos de prueba.
3. Desplegar API con el flujo nuevo. El endpoint viejo deja de existir en su forma actual, por lo que la app de riders debe salir coordinada.
4. Habilitar `Rider` en los endpoints de Pago Móvil y comunicar a los riders activos que registren sus datos — sin ellos, la solicitud falla con `pago_movil_required`.
5. CRM consume el listado nuevo.

**Rollback:** el `-- +goose Down` revierte el reetiquetado de movimientos a `withdrawal` y elimina la tabla e índices. Los movimientos de ciclo nuevos (`withdrawal_hold`, `withdrawal_processing`, `withdrawal_restore`) no tienen equivalente en el esquema viejo: un rollback con solicitudes abiertas dejaría holds sin solicitud que los explique, así que antes de revertir hay que resolver (pagar o rechazar) toda solicitud activa.

## Open Questions

Ninguna. Las dos que había —el tratamiento del backfill y la coordinación con la app de riders— se resuelven con el mismo hecho: la plataforma todavía no opera, así que no hay dinero real ni clientes consumiendo el endpoint.
