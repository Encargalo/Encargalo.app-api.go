## Purpose

Define la operación del CRM sobre las solicitudes de retiro: cómo staff las encuentra, cómo registra el pago con su comprobante y cómo las rechaza dejando trazabilidad de quién hizo qué.

## ADDED Requirements

### Requirement: Listado de solicitudes de retiro para staff

El sistema SHALL exponer a usuarios con rol `Staff` el listado de solicitudes de retiro de riders mediante `GET /riders/staff/finance/withdrawals`, filtrable por estado y ordenado por fecha de creación descendente. El listado SHALL incluir, por solicitud: identificador, monto solicitado en Bs y su equivalente en USD a la tasa vigente, **monto pagable actual** (el neto, o el congelado si la solicitud ya fue reclamada), estado, nombre, apellido y teléfono del rider, los datos de Pago Móvil congelados en la solicitud, y las fechas de las transiciones ocurridas.

El monto pagable SHALL presentarse de forma que sea inequívocamente el valor a transferir, distinguible del monto solicitado originalmente, para que un operador no transfiera el monto solicitado cuando el neto es menor.

Una solicitud de retiro SHALL ser visible en este listado independientemente del balance actual del rider. El listado de payables (`GET /riders/staff/finance/payables`), que solo muestra entidades con saldo positivo, NO SHALL ser el mecanismo por el cual staff descubre retiros pendientes.

#### Scenario: Listado de pendientes

- **WHEN** staff consulta `GET /riders/staff/finance/withdrawals?status=pending`
- **THEN** el sistema SHALL retornar HTTP 200 con las solicitudes en estado `pending`, incluyendo las de riders cuyo balance quedó en 0 por la propia retención

#### Scenario: Listado sin filtro de estado

- **WHEN** staff consulta `GET /riders/staff/finance/withdrawals` sin parámetro `status`
- **THEN** el sistema SHALL retornar las solicitudes activas (`pending` y `processing`), por ser las que requieren acción operativa

#### Scenario: Filtro por estado resuelto

- **WHEN** staff consulta el listado filtrando por `completed` o `rejected`
- **THEN** el sistema SHALL retornar esas solicitudes con su comprobante o su motivo de rechazo respectivamente

#### Scenario: Acceso sin rol Staff

- **WHEN** un usuario sin rol `Staff` consulta el listado
- **THEN** el sistema SHALL rechazar con HTTP 401

### Requirement: Reclamo de la solicitud antes de transferir

El sistema SHALL exigir que staff reclame una solicitud mediante `POST /riders/staff/finance/withdrawals/:id/claim` antes de poder registrar su pago. El reclamo SHALL pasar la solicitud a `processing`, calcular el monto pagable neto, escribirlo en la solicitud como monto congelado, y registrar el operador que la reclamó junto con la fecha.

El reclamo existe porque la transferencia bancaria ocurre fuera del sistema: entre que el operador ve el monto y registra el pago pasan minutos en los que el balance del rider puede moverse. Congelar al reclamar SHALL hacer vinculante el monto que el operador vio y transfirió.

#### Scenario: Reclamo exitoso

- **WHEN** staff reclama una solicitud en estado `pending`
- **THEN** el sistema SHALL pasarla a `processing`, congelar en ella el monto pagable neto, y retornar ese monto como el valor a transferir

#### Scenario: Reclamo de una solicitud ya reclamada

- **WHEN** staff reclama una solicitud que ya está en `processing`
- **THEN** el sistema SHALL rechazar con HTTP 409 y código `withdrawal_invalid_transition`, de modo que dos operadores no transfieran la misma solicitud

#### Scenario: Reclamo que resulta en monto cero

- **WHEN** staff reclama una solicitud cuyo monto pagable neto es cero o negativo
- **THEN** el sistema SHALL anular la solicitud e informar que no hay nada que transferir, sin pasarla a `processing`

### Requirement: Registro del pago con comprobante obligatorio

El sistema SHALL permitir a staff registrar el pago de una solicitud reclamada mediante `POST /riders/staff/finance/withdrawals/:id/complete`. El comprobante de la transferencia SHALL ser obligatorio: el sistema NO SHALL completar una solicitud sin él. La solicitud SHALL pasar a estado `completed`, registrar el identificador del staff que la pagó, la URL del comprobante y la fecha de completado, y registrar el movimiento `withdrawal_payout` que convierte la retención en pago efectivo.

El monto pagado SHALL ser el monto congelado al reclamar, y el sistema NO SHALL recalcularlo al registrar el pago: la transferencia ya ocurrió por ese valor. El sistema NO SHALL exigir que el balance actual del solicitante cubra el pago. Cualquier diferencia entre lo transferido y el monto congelado SHALL corregirse como un ajuste de saldo independiente, con su propio motivo, y NO alterando el monto de la solicitud.

#### Scenario: Pago exitoso

- **WHEN** staff completa una solicitud en estado `processing` adjuntando el comprobante de la transferencia
- **THEN** el sistema SHALL almacenar el comprobante, pasar la solicitud a `completed` con `staff_id` y `completed_at`, registrar el movimiento `withdrawal_payout` por el monto congelado, e iniciar el cooldown de 24 horas del rider desde ese momento

#### Scenario: Pago sin comprobante

- **WHEN** staff intenta completar una solicitud sin adjuntar comprobante
- **THEN** el sistema SHALL rechazar con HTTP 400 y código `withdrawal_proof_required`, y la solicitud SHALL permanecer en su estado anterior

#### Scenario: Comprobante que no se puede almacenar

- **WHEN** el almacenamiento del comprobante falla
- **THEN** el sistema SHALL rechazar la operación y la solicitud SHALL permanecer en su estado anterior, sin movimiento de pago registrado

#### Scenario: Pago de una solicitud ya resuelta

- **WHEN** staff intenta completar una solicitud en estado `completed` o `rejected`
- **THEN** el sistema SHALL rechazar con HTTP 409 y código `withdrawal_invalid_transition`

#### Scenario: El balance en 0 no bloquea el pago

- **WHEN** staff completa la solicitud de un rider cuyo balance disponible es 0 porque el monto está retenido en esa misma solicitud
- **THEN** el pago SHALL registrarse correctamente, sin error de saldo insuficiente

#### Scenario: El balance se movió entre el reclamo y el pago

- **WHEN** staff reclamó una solicitud por 800 Bs y, antes de registrar el pago, el rider completó un pedido en efectivo que le debitó 728 Bs
- **THEN** el pago SHALL registrarse por los 800 Bs congelados al reclamar, y el saldo del rider SHALL quedar en −728 Bs como deuda visible, sin que el sistema altere el monto pagado

### Requirement: Rechazo con motivo

El sistema SHALL permitir a staff rechazar una solicitud mediante `POST /riders/staff/finance/withdrawals/:id/reject`. El motivo del rechazo SHALL ser obligatorio. La solicitud SHALL pasar a estado `rejected` registrando el identificador del staff, el motivo y la fecha de resolución, y el saldo retenido SHALL devolverse al solicitante.

#### Scenario: Rechazo exitoso

- **WHEN** staff rechaza una solicitud `pending` indicando el motivo
- **THEN** el sistema SHALL pasarla a `rejected` con `staff_id`, `rejection_reason` y `resolved_at`, devolver el saldo retenido, y el motivo SHALL quedar visible para el rider

#### Scenario: Rechazo sin motivo

- **WHEN** staff intenta rechazar una solicitud sin indicar motivo
- **THEN** el sistema SHALL rechazar con HTTP 400 y código `withdrawal_rejection_reason_required`, y la solicitud SHALL permanecer en su estado anterior

### Requirement: Trazabilidad del pago en el historial

Los movimientos generados por el ciclo de retiro SHALL aparecer en el historial de movimientos de la entidad (`GET /riders/transactions` y `GET /riders/staff/finance/:riderId/transactions`), y el movimiento de pago SHALL exponer el comprobante y el staff responsable.

#### Scenario: Historial de un retiro pagado

- **WHEN** se consulta el historial de un rider con un retiro ya pagado
- **THEN** el historial SHALL incluir el movimiento `withdrawal_payout` con la URL del comprobante y el identificador del staff que registró el pago

### Requirement: Aviso de solicitud nueva

Al crearse una solicitud de retiro, el sistema SHALL emitir un evento que derive en una notificación al canal operativo del equipo, para que una solicitud no quede sin atender por no haber nadie mirando el CRM.

#### Scenario: Notificación al solicitar

- **WHEN** un rider crea una solicitud de retiro
- **THEN** el sistema SHALL publicar el evento correspondiente con el identificador de la solicitud, el rider y el monto, y una falla en la notificación NO SHALL impedir que la solicitud se cree
