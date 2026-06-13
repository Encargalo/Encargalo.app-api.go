## ADDED Requirements

### Requirement: El rider tiene un balance financiero acumulado

Cada rider SHALL tener un balance en USD que represente la diferencia acumulada entre el efectivo cobrado directamente a clientes y el dinero recibido por la plataforma en su nombre.

- Balance negativo → el rider tiene en su poder dinero de la empresa.
- Balance positivo → la empresa le debe dinero al rider.

El balance se ajusta automáticamente al completar cada carrera según el tipo de pago.

#### Scenario: Carrera completada con pago en efectivo

- **WHEN** un pedido transiciona a `Completed` y el tipo de pago es `CashBs` o `CashUSD`
- **THEN** el balance del rider SHALL decrementarse en la ganancia neta del rider (`monto_carrera × 0.80`)

#### Scenario: Carrera completada con pago bancario

- **WHEN** un pedido transiciona a `Completed` y el tipo de pago es bancario
- **THEN** el balance del rider SHALL incrementarse en la ganancia neta del rider (`monto_carrera × 0.80`)

---

### Requirement: Cada ajuste de balance genera un movimiento trazable

El sistema SHALL registrar un movimiento por cada ajuste de balance, vinculado al pedido que lo originó.

Campos del movimiento: `order_id`, `amount` (monto de carrera antes de comisión), `net_amount` (ganancia rider), `commission` (20%), `payment_method`, `tariff_rate`, `distance_km`, `timestamp`.

#### Scenario: Movimiento registrado al completar carrera

- **WHEN** el balance del rider se ajusta al completar un pedido
- **THEN** el sistema SHALL crear un registro en `riders.balance_movements` con todos los campos requeridos

#### Scenario: Consulta de balance actual

- **WHEN** se realiza `GET /api/v1/riders/:id/balance`
- **THEN** el sistema SHALL devolver el balance actual del rider y los últimos movimientos (paginados, máximo 20 por página)

#### Scenario: Consulta de historial de transacciones

- **WHEN** se realiza `GET /api/v1/riders/:id/transactions`
- **THEN** el sistema SHALL devolver el historial completo de movimientos paginado, ordenado por timestamp descendente

---

### Requirement: La comisión de la plataforma es del 20% por carrera

Por cada carrera completada, el sistema SHALL calcular y registrar:

- `ganancia_rider = monto_carrera × 0.80`
- `comision_plataforma = monto_carrera × 0.20`

El ajuste al balance del rider MUST usar `ganancia_rider`, no `monto_carrera`.

#### Scenario: Desglose correcto en movimiento bancario

- **WHEN** se registra un movimiento de tipo bancario con `monto_carrera = $5.00`
- **THEN** `net_amount` SHALL ser `$4.00` y `commission` SHALL ser `$1.00`

#### Scenario: Desglose correcto en movimiento en efectivo

- **WHEN** se registra un movimiento de tipo efectivo con `monto_carrera = $5.00`
- **THEN** `net_amount` SHALL ser `$4.00`, `commission` SHALL ser `$1.00`, y el balance del rider SHALL decrementarse en `$4.00`
