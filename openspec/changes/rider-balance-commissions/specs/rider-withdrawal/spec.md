## ADDED Requirements

### Requirement: Solicitud de retiro manual
El sistema SHALL permitir a un rider autenticado solicitar retiro de su saldo disponible mediante `POST /riders/withdrawal` cuando su balance sea ≥ +$15. El retiro descuenta el monto del balance y registra una transacción de tipo `withdrawal`.

El monto retirado corresponde al balance completo disponible en el momento de la solicitud (retiro total, no parcial).

#### Scenario: Retiro exitoso con balance suficiente
- **WHEN** un rider con balance ≥ +$15 hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL registrar una transacción con `movement_type = 'withdrawal'` y `amount` negativo igual al balance actual, decrementar el balance a 0, y retornar HTTP 201 con el monto retirado

#### Scenario: Retiro rechazado por balance insuficiente
- **WHEN** un rider con balance < +$15 hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL retornar HTTP 422 con mensaje `"balance insuficiente para solicitar retiro (mínimo $15)"`

#### Scenario: Retiro rechazado con balance negativo
- **WHEN** un rider con balance negativo hace `POST /riders/withdrawal`
- **THEN** el sistema SHALL retornar HTTP 422 con mensaje `"balance insuficiente para solicitar retiro (mínimo $15)"`

#### Scenario: Concurrencia en retiro
- **WHEN** dos solicitudes de retiro del mismo rider llegan simultáneamente
- **THEN** solo una SHALL procesarse exitosamente; la segunda retorna HTTP 422 porque el balance ya fue descontado
