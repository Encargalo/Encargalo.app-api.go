## ADDED Requirements

### Requirement: Zonas de balance del rider
El sistema SHALL clasificar el balance de cada rider en tres zonas que determinan el comportamiento de asignación:

| Zona | Condición | Comportamiento |
|---|---|---|
| `risk` | balance ≤ −$15 | Solo puede aceptar carreras con pago bancario |
| `normal` | −$15 < balance < +$15 | Puede aceptar cualquier carrera |
| `withdrawal_available` | balance ≥ +$15 | Puede aceptar cualquier carrera y solicitar retiro |

#### Scenario: Rider en zona normal acepta carrera de efectivo
- **WHEN** un rider con balance en zona `normal` intenta aceptar una orden con pago en efectivo
- **THEN** el sistema permite la asignación normalmente

#### Scenario: Rider en zona normal acepta carrera bancaria
- **WHEN** un rider con balance en zona `normal` intenta aceptar una orden con pago bancario
- **THEN** el sistema permite la asignación normalmente

### Requirement: Bloqueo de carreras de efectivo en zona de riesgo
El sistema SHALL rechazar la aceptación de órdenes con pago en efectivo cuando el rider esté en zona de riesgo (balance ≤ −$15). La validación ocurre en `AcceptOrder` antes de la asignación.

#### Scenario: Rider en zona de riesgo intenta aceptar carrera de efectivo
- **WHEN** un rider con balance ≤ −$15 envía `{"type":"accept_order","order_id":"..."}` para una orden con `method_payment = 'cash'`
- **THEN** el sistema NO asigna la orden y envía al rider `{"type":"error","message":"solo puedes aceptar carreras con pago bancario"}`

#### Scenario: Rider en zona de riesgo acepta carrera bancaria
- **WHEN** un rider con balance ≤ −$15 envía `{"type":"accept_order","order_id":"..."}` para una orden con pago bancario
- **THEN** el sistema asigna la orden normalmente

#### Scenario: Rider sale de zona de riesgo al completar carrera bancaria
- **WHEN** un rider en zona de riesgo completa una carrera bancaria y su balance sube por encima de −$15
- **THEN** en la siguiente orden puede aceptar carreras de efectivo nuevamente

### Requirement: Verificación de balance con consistencia transaccional
La verificación del balance y la asignación de la orden SHALL ejecutarse dentro de una transacción SQL para evitar race conditions.

#### Scenario: Dos riders intentan aceptar la misma orden simultáneamente
- **WHEN** dos riders envían `accept_order` para la misma orden al mismo tiempo
- **THEN** solo uno recibe `order_accepted`; el otro recibe `order_taken`
