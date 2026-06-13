## ADDED Requirements

### Requirement: Los umbrales de balance controlan qué carreras puede recibir un rider

El sistema SHALL evaluar el balance del rider antes de asignarle una carrera y aplicar las siguientes restricciones:

| Estado | Condición | Comportamiento |
|---|---|---|
| Zona de riesgo | `balance ≤ -$15` | Solo carreras con pago bancario |
| Zona normal | `-$15 < balance < +$15` | Asignación sin restricción |
| Retiro disponible | `balance ≥ +$15` | Asignación normal + retiro habilitado |

#### Scenario: Rider en zona de riesgo recibe solo carreras bancarias

- **WHEN** el balance de un rider es `≤ -$15.00` y se intenta asignarle una carrera con pago en efectivo
- **THEN** el sistema SHALL rechazar la asignación con error `ErrRiderBalanceRestricted`

#### Scenario: Rider en zona de riesgo puede recibir carrera bancaria

- **WHEN** el balance de un rider es `≤ -$15.00` y se intenta asignarle una carrera con pago bancario
- **THEN** el sistema SHALL permitir la asignación normalmente

#### Scenario: Rider vuelve a zona normal

- **WHEN** el balance de un rider pasa de `≤ -$15` a `> -$5` (por completar carreras bancarias)
- **THEN** el sistema SHALL restaurar la asignación sin restricción para todas las modalidades de pago

#### Scenario: Rider en zona normal recibe cualquier carrera

- **WHEN** el balance de un rider está entre `-$15` y `+$15`
- **THEN** el sistema SHALL asignar la carrera independientemente del tipo de pago
