## ADDED Requirements

### Requirement: Cálculo de monto de carrera al completar una orden
Al transicionar una orden al estado `Completed`, el sistema SHALL calcular automáticamente el monto de la carrera del rider usando Google Maps Distance Matrix API (distancia en coche desde el shop hasta la dirección del cliente) y la tarifa correspondiente a la hora de creación de la orden.

Tarifas:
- Diurna (creación antes de las 22:00): `$0.40 / km`
- Nocturna (creación desde las 22:00): `$0.60 / km`

Fórmula: `monto_carrera = distancia_km × tarifa`

Comisión de la plataforma (20%):
- `ganancia_rider = monto_carrera × 0.80`
- `comision_plataforma = monto_carrera × 0.20`

#### Scenario: Orden completada en horario diurno
- **WHEN** una orden transiciona a `Completed` y fue creada antes de las 22:00
- **THEN** el sistema calcula `monto_carrera = distancia_km × 0.40` y registra `ganancia_rider = monto_carrera × 0.80`

#### Scenario: Orden completada en horario nocturno
- **WHEN** una orden transiciona a `Completed` y fue creada desde las 22:00
- **THEN** el sistema calcula `monto_carrera = distancia_km × 0.60` y registra `ganancia_rider = monto_carrera × 0.80`

#### Scenario: Fallo de Google Maps al completar orden
- **WHEN** la API de Google Maps Distance Matrix retorna error al intentar calcular la distancia
- **THEN** el sistema SHALL loguear el error, completar la transición de estado de la orden sin bloquearla, y registrar la transacción con `distance_km = 0`

### Requirement: Ajuste de balance según tipo de pago al completar carrera
Al registrar la ganancia de una carrera completada, el sistema SHALL ajustar el balance del rider en `finance.balances` según el tipo de pago de la orden.

- Pago en efectivo (`cash`): `balance -= ganancia_rider` (el rider tiene ese dinero en su poder)
- Pago bancario (`bank`): `balance += ganancia_rider` (la plataforma recibió el dinero y se lo debe al rider)

#### Scenario: Carrera completada con pago en efectivo
- **WHEN** una orden con `method_payment = 'cash'` transiciona a `Completed`
- **THEN** el balance del rider DEBE decrementarse en `ganancia_rider` y se registra una transacción con `movement_type = 'ride_cash'` y `amount` negativo

#### Scenario: Carrera completada con pago bancario
- **WHEN** una orden con `method_payment != 'cash'` transiciona a `Completed`
- **THEN** el balance del rider DEBE incrementarse en `ganancia_rider` y se registra una transacción con `movement_type = 'ride_bank'` y `amount` positivo

### Requirement: Endpoint GET /riders/balance
El sistema SHALL exponer `GET /riders/balance` (autenticación requerida, rol `Rider`) que retorne el balance actual del rider autenticado y sus últimos 10 movimientos.

Respuesta:
```json
{
  "balance": -12.50,
  "zone": "risk",
  "transactions": [
    {
      "id": "uuid",
      "order_id": "uuid",
      "amount": -8.00,
      "movement_type": "ride_cash",
      "payment_method": "cash",
      "distance_km": 3.2,
      "created_at": "2026-06-14T20:00:00Z"
    }
  ]
}
```

Campo `zone`: `"risk"` (balance ≤ −15), `"normal"` (−15 < balance < 15), `"withdrawal_available"` (balance ≥ 15).

#### Scenario: Rider consulta su balance
- **WHEN** un rider autenticado hace `GET /riders/balance`
- **THEN** el sistema retorna HTTP 200 con el balance actual, la zona y los últimos 10 movimientos

#### Scenario: Rider sin movimientos consulta su balance
- **WHEN** un rider autenticado hace `GET /riders/balance` y no tiene movimientos registrados
- **THEN** el sistema retorna HTTP 200 con `balance = 0`, `zone = "normal"` y `transactions = []`
