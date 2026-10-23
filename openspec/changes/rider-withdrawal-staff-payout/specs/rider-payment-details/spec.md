## Purpose

Define de dónde sale el destino de un retiro: los datos de Pago Móvil de la cuenta del rider, su captura y la copia que cada solicitud congela para que la historia no se reescriba cuando el rider los cambia.

## ADDED Requirements

### Requirement: El rider administra sus datos de Pago Móvil

El sistema SHALL permitir a un usuario con rol `Rider` consultar y registrar sus datos de Pago Móvil (teléfono, código de banco y cédula) sobre su propia cuenta, mediante los mismos endpoints `GET /users/pago-movil` y `PUT /users/pago-movil` que usan los clientes. Los datos SHALL vivir en la cuenta del usuario y no duplicarse en el perfil de rider.

#### Scenario: Rider registra su Pago Móvil

- **WHEN** un usuario con rol `Rider` hace `PUT /users/pago-movil` con teléfono, código de banco y cédula válidos
- **THEN** el sistema SHALL guardarlos en su cuenta y retornar HTTP 200 con los datos registrados

#### Scenario: Rider consulta su Pago Móvil

- **WHEN** un usuario con rol `Rider` hace `GET /users/pago-movil` teniendo datos registrados
- **THEN** el sistema SHALL retornar HTTP 200 con esos datos

#### Scenario: Rider sin datos registrados

- **WHEN** un usuario con rol `Rider` hace `GET /users/pago-movil` sin datos registrados
- **THEN** el sistema SHALL responder HTTP 404

#### Scenario: Acceso de clientes preservado

- **WHEN** un usuario con rol `Customer` opera sobre `GET` o `PUT /users/pago-movil`
- **THEN** el sistema SHALL seguir atendiéndolo como antes de este cambio

### Requirement: Los datos de pago se congelan en cada solicitud

Cada solicitud de retiro SHALL guardar una copia de los datos de Pago Móvil vigentes al momento de crearse. El sistema SHALL mostrar a staff esa copia congelada y NO los datos actuales de la cuenta, de modo que el comprobante de un pago corresponda siempre a la cuenta a la que efectivamente se transfirió.

#### Scenario: Cambio de datos no altera una solicitud abierta

- **WHEN** un rider crea una solicitud de retiro y después cambia su teléfono de Pago Móvil
- **THEN** la solicitud SHALL conservar y mostrar a staff el teléfono con el que se creó

#### Scenario: Cambio de datos no altera solicitudes pagadas

- **WHEN** un rider cambia sus datos de Pago Móvil después de que un retiro fue pagado
- **THEN** la solicitud `completed` SHALL conservar los datos con los que se hizo esa transferencia
