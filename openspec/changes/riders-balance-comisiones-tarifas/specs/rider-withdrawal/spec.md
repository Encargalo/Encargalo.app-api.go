## ADDED Requirements

### Requirement: El rider puede solicitar un retiro cuando su balance es suficiente

El sistema SHALL permitir al rider solicitar un retiro cuando su balance es `≥ +$15.00`. El retiro no es automático; siempre es iniciado por el rider.

Al procesar el retiro, el balance SHALL decrementarse por el monto retirado.

#### Scenario: Solicitud de retiro aprobada

- **WHEN** se realiza `POST /api/v1/riders/:id/withdrawal` y el balance es `≥ +$15.00`
- **THEN** el sistema SHALL registrar la solicitud de retiro, decrementar el balance por el saldo disponible y devolver `200` con el monto retirado y el nuevo balance

#### Scenario: Solicitud de retiro rechazada por balance insuficiente

- **WHEN** se realiza `POST /api/v1/riders/:id/withdrawal` y el balance es `< +$15.00`
- **THEN** el sistema SHALL rechazar la solicitud con `422` y error `ErrInsufficientBalance`

#### Scenario: El retiro descuenta el saldo completo disponible

- **WHEN** el balance del rider es `+$23.50` y solicita retiro
- **THEN** el retiro SHALL ser por `$23.50` y el nuevo balance SHALL ser `$0.00`

---

### Requirement: Solo el propio rider puede solicitar su retiro

El endpoint `POST /api/v1/riders/:id/withdrawal` SHALL requerir autenticación y verificar que el rider autenticado es el dueño del recurso solicitado.

#### Scenario: Rider solicita su propio retiro

- **WHEN** el rider autenticado realiza `POST /api/v1/riders/:id/withdrawal` donde `:id` coincide con su propia identidad
- **THEN** el sistema SHALL procesar la solicitud

#### Scenario: Rider intenta retirar balance de otro rider

- **WHEN** el rider autenticado realiza `POST /api/v1/riders/:id/withdrawal` donde `:id` es de otro rider
- **THEN** el sistema SHALL rechazar con `401` y error `ErrUnautorized`
