## Purpose

Define el ciclo de vida de una solicitud de retiro de saldo: cómo nace, qué invariantes la protegen, cómo se refleja en el ledger y cómo vuelve el dinero al solicitante si no se le paga. Es genérica por tipo de entidad para que riders y comercios compartan la misma política.

## ADDED Requirements

### Requirement: Creación de una solicitud de retiro

El sistema SHALL permitir a un rider autenticado solicitar el retiro de su saldo mediante `POST /riders/withdrawal`. La solicitud SHALL crearse en estado `pending`, retener el saldo del solicitante y quedar registrada como deuda pendiente de la plataforma. El sistema NO SHALL considerar el retiro pagado al crearlo: sin un pago registrado por staff, la solicitud permanece abierta indefinidamente.

El monto solicitado SHALL ser el balance completo disponible al momento de solicitar (retiro total, no parcial). El saldo que la entidad acumule mientras la solicitud está abierta SHALL quedar disponible para una solicitud posterior y NO SHALL alterar el monto de la solicitud en curso.

#### Scenario: Solicitud exitosa

- **WHEN** un rider con balance ≥ al equivalente de $3 según la tasa BCV vigente, con Pago Móvil registrado, sin solicitud activa y sin cooldown vigente hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL crear una solicitud en estado `pending` con el monto igual al balance, congelar en ella los datos de Pago Móvil del solicitante, registrar un movimiento `withdrawal_hold` con `amount` negativo igual al balance, dejar el balance en 0, y retornar HTTP 201 con el identificador de la solicitud, su monto, su estado y los datos de pago congelados

#### Scenario: Balance por debajo del mínimo

- **WHEN** un rider con balance menor al equivalente de $3 según la tasa BCV vigente hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL rechazar con HTTP 422 y código `insufficient_balance`, sin crear solicitud ni movimiento alguno

#### Scenario: Balance negativo

- **WHEN** un rider con balance negativo hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL rechazar con HTTP 422 y código `insufficient_balance`

#### Scenario: Ya existe una solicitud activa

- **WHEN** un rider con una solicitud en estado `pending` o `processing` hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL rechazar con HTTP 409 y código `withdrawal_already_requested`, sin crear una segunda solicitud

#### Scenario: Cooldown vigente

- **WHEN** un rider cuyo último retiro `completed` se pagó hace menos de 24 horas hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL rechazar con HTTP 429 y código `withdrawal_cooldown`

#### Scenario: Sin datos de pago registrados

- **WHEN** un rider sin Pago Móvil registrado hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL rechazar con HTTP 422 y código `pago_movil_required`, sin retener saldo

#### Scenario: Tasa BCV no disponible

- **WHEN** no hay tasa BCV vigente y un rider hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL rechazar con HTTP 503 y código `bcv_rate_unavailable`, porque el mínimo de retiro no puede evaluarse sin tasa

#### Scenario: Dos solicitudes simultáneas del mismo rider

- **WHEN** dos solicitudes de retiro del mismo rider llegan de forma concurrente
- **THEN** exactamente una SHALL crearse y la otra SHALL rechazarse con HTTP 409 y código `withdrawal_already_requested`, garantizado por una restricción de unicidad en el almacenamiento y no solo por validación en memoria

### Requirement: Una sola solicitud activa por entidad

El sistema SHALL garantizar que una entidad no tenga más de una solicitud de retiro en estado `pending` o `processing` al mismo tiempo. Esta garantía SHALL ser estructural en el almacenamiento, de modo que dos escrituras concurrentes no puedan violarla.

#### Scenario: Solicitud nueva después de una resuelta

- **WHEN** la solicitud anterior de una entidad está en estado `completed` o `rejected`
- **THEN** la restricción de solicitud activa SHALL permitir crear una solicitud nueva, sujeta a las demás reglas (cooldown, mínimo, datos de pago)

### Requirement: Estados y transiciones

Una solicitud de retiro SHALL tener exactamente uno de los estados `pending`, `processing`, `completed` o `rejected`. El sistema SHALL permitir únicamente las transiciones `pending → processing`, `processing → completed`, `pending → rejected` y `processing → rejected`. Los estados `completed` y `rejected` SHALL ser terminales.

El sistema NO SHALL permitir `pending → completed` directo: toda solicitud SHALL pasar por `processing`, porque es ese paso el que congela el monto que se va a transferir.

Cada transición SHALL registrar su propio movimiento en el ledger: `withdrawal_hold` al crearse, `withdrawal_processing` al pasar a pago en curso, `withdrawal_payout` al completarse y `withdrawal_restore` al rechazarse. El sistema SHALL garantizar que una misma solicitud no acumule más de una reserva de saldo, de modo que el saldo no pueda retenerse ni liberarse dos veces por el mismo retiro.

#### Scenario: Transición inválida

- **WHEN** se intenta llevar una solicitud en estado `completed` o `rejected` a cualquier otro estado
- **THEN** el sistema SHALL rechazar con HTTP 409 y código `withdrawal_invalid_transition`, y la solicitud SHALL conservar su estado y sus movimientos

#### Scenario: Pago sin reclamo previo

- **WHEN** se intenta completar una solicitud que está en estado `pending`
- **THEN** el sistema SHALL rechazar con HTTP 409 y código `withdrawal_invalid_transition`, porque no existe un monto congelado que transferir

### Requirement: El monto solicitado es un techo, no una promesa

El monto registrado al crear la solicitud SHALL ser el saldo retenido en ese momento. El monto efectivamente pagable SHALL calcularse al reclamar la solicitud como el saldo retenido más la porción negativa del saldo disponible, de modo que la plataforma nunca transfiera más de lo que neto adeuda. Una vez reclamada, ese monto SHALL quedar congelado y NO SHALL recalcularse al registrar el pago.

#### Scenario: Efectivo cobrado antes del reclamo reduce lo pagable

- **WHEN** un rider solicita un retiro de 800 Bs y después completa un pedido en efectivo que le debita 728 Bs, y staff reclama la solicitud
- **THEN** el monto pagable congelado SHALL ser 72 Bs, y al pagarlo la deuda de 728 Bs SHALL quedar saldada contra el propio retiro

#### Scenario: Ganancias posteriores no aumentan lo pagable

- **WHEN** un rider solicita un retiro de 800 Bs, después gana 150 Bs por pagos bancarios, y staff reclama la solicitud
- **THEN** el monto pagable SHALL ser 800 Bs y los 150 Bs SHALL permanecer disponibles para una solicitud posterior

#### Scenario: El efectivo consume el retiro completo

- **WHEN** al reclamar una solicitud el monto pagable resulta cero o negativo porque el saldo disponible quedó más negativo que lo retenido
- **THEN** el sistema SHALL anular la solicitud sin transferencia, el rider SHALL conservar su deuda en el balance, y el cooldown de 24 horas NO SHALL activarse

#### Scenario: Efectivo cobrado después del reclamo

- **WHEN** staff reclama una solicitud por 800 Bs y el rider completa un pedido en efectivo que le debita 728 Bs antes de que staff registre el pago
- **THEN** el pago SHALL registrarse por los 800 Bs congelados al reclamar, y la deuda de 728 Bs SHALL quedar reflejada como saldo negativo del rider, saldable contra ganancias futuras

#### Scenario: Solicitud inexistente

- **WHEN** se opera sobre un identificador de solicitud que no existe
- **THEN** el sistema SHALL responder HTTP 404 y código `withdrawal_not_found`

### Requirement: Restauración de saldo al rechazar

Cuando una solicitud se rechaza, el sistema SHALL devolver al solicitante el monto retenido mediante un movimiento `withdrawal_restore` de signo positivo, dejando el balance como estaba antes de la solicitud más cualquier movimiento ocurrido mientras estaba abierta. El rechazo NO SHALL iniciar el cooldown de 24 horas.

#### Scenario: Rechazo devuelve el saldo

- **WHEN** staff rechaza una solicitud de 800 Bs de un rider cuyo balance actual es 150 Bs (acumulados mientras el retiro estaba abierto)
- **THEN** el sistema SHALL registrar un movimiento `withdrawal_restore` de +800 Bs, el balance SHALL quedar en 950 Bs, y el rider SHALL poder solicitar un retiro nuevo de inmediato sin esperar 24 horas

### Requirement: Cooldown anclado en el pago completado

El sistema SHALL impedir una solicitud nueva si han pasado menos de 24 horas desde `completed_at` de la última solicitud de la entidad en estado `completed`. El cooldown NO SHALL medirse desde la fecha de creación de la solicitud, y las solicitudes en estado `rejected` NO SHALL contar para el cálculo.

#### Scenario: Pago rápido no adelanta el reloj

- **WHEN** un rider solicita a las 09:00 del lunes y staff le paga a las 21:00 del lunes
- **THEN** la siguiente solicitud SHALL rechazarse hasta las 21:00 del martes, 24 horas después del pago y no de la solicitud

#### Scenario: Rechazo no consume el cooldown

- **WHEN** la última solicitud de un rider fue rechazada hace una hora y no tiene ninguna solicitud `completed` en las últimas 24 horas
- **THEN** el sistema SHALL permitir crear una solicitud nueva

### Requirement: Visibilidad de la solicitud para el solicitante

El rider SHALL poder ver el estado de su retiro en curso. `GET /riders/balance` SHALL informar la solicitud activa (identificador, monto y estado) cuando exista, de modo que un balance en 0 por un retiro pendiente sea distinguible de un balance en 0 por no haber ganado nada.

#### Scenario: Balance con retiro pendiente

- **WHEN** un rider con una solicitud `pending` de 800 Bs consulta `GET /riders/balance`
- **THEN** la respuesta SHALL incluir la solicitud activa con su monto y estado, además del balance disponible

#### Scenario: Balance sin retiro pendiente

- **WHEN** un rider sin solicitudes activas consulta `GET /riders/balance`
- **THEN** la respuesta SHALL indicar la ausencia de solicitud activa
