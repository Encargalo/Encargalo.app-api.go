## MODIFIED Requirements

### Requirement: Completar un pedido actualiza el balance del rider asignado

Cuando un pedido transiciona al estado `Completed` (vía `ConfirmDelivery`), el sistema SHALL:

1. Determinar la tarifa por km (diurna/nocturna) según `order.created_at`.
2. Calcular `monto_carrera = order.distance_km × tarifa`.
3. Calcular `ganancia_rider = monto_carrera × 0.80`.
4. Ajustar el balance del rider según el tipo de pago:
   - Efectivo → `balance -= ganancia_rider`
   - Bancario → `balance += ganancia_rider`
5. Registrar el movimiento en `riders.balance_movements`.

Si el rider asignado no está disponible o el cálculo falla, el pedido MUST transicionar a `Completed` de todas formas; el error de balance se registra en logs sin bloquear la transición.

#### Scenario: Pedido bancario completado — balance incrementa

- **WHEN** `ConfirmDelivery` se ejecuta exitosamente y el tipo de pago es bancario
- **THEN** el balance del rider SHALL incrementarse en `ganancia_rider` y SHALL existir un movimiento asociado al `order_id`

#### Scenario: Pedido en efectivo completado — balance decrementa

- **WHEN** `ConfirmDelivery` se ejecuta exitosamente y el tipo de pago es en efectivo
- **THEN** el balance del rider SHALL decrementarse en `ganancia_rider` y SHALL existir un movimiento asociado al `order_id`

#### Scenario: Fallo en cálculo de balance no bloquea la completación

- **WHEN** `ConfirmDelivery` se ejecuta pero el cálculo de balance falla (e.g., `distance_km` no disponible)
- **THEN** el pedido SHALL transicionar a `Completed` de todas formas y el error SHALL quedar registrado en los logs del sistema
