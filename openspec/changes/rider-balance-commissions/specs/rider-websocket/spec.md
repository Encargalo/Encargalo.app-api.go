## MODIFIED Requirements

### Requirement: Mensaje new_order incluye datos de ganancia
El mensaje `new_order` enviado por el servidor al rider SHALL incluir los campos `rider_earnings` y, cuando el tipo de pago es en efectivo, `collect_amount`.

Estructura actualizada del campo `order` en `new_order`:
```json
{
  "type": "new_order",
  "order": {
    "id": "uuid",
    "status": "Pending",
    "address": "...",
    "method_payment": "cash",
    "total_price": 25000,
    "delivery_fee": 5000,
    "rider_earnings": 3.20,
    "collect_amount": 25000
  }
}
```

- `rider_earnings`: ganancia estimada del rider en la carrera (en la moneda de la plataforma).
- `collect_amount`: monto que el rider debe cobrar al cliente. Solo se incluye cuando `method_payment = 'cash'`. Para pagos bancarios este campo NO se incluye.

#### Scenario: Rider recibe new_order con pago en efectivo
- **WHEN** el servidor hace broadcast de una nueva orden con `method_payment = 'cash'`
- **THEN** el mensaje incluye `rider_earnings` y `collect_amount` igual al `total_price` de la orden

#### Scenario: Rider recibe new_order con pago bancario
- **WHEN** el servidor hace broadcast de una nueva orden con pago bancario
- **THEN** el mensaje incluye `rider_earnings` pero NO incluye `collect_amount`

### Requirement: Mensaje order_accepted incluye datos de ganancia
El mensaje `order_accepted` enviado al rider SHALL incluir `rider_earnings` y, cuando corresponda, `collect_amount`.

```json
{
  "type": "order_accepted",
  "order": {
    "id": "uuid",
    "pickup_code": "482931",
    "method_payment": "cash",
    "rider_earnings": 3.20,
    "collect_amount": 25000
  }
}
```

#### Scenario: Rider recibe order_accepted con pago en efectivo
- **WHEN** el rider acepta exitosamente una orden con pago en efectivo
- **THEN** el mensaje `order_accepted` incluye `pickup_code`, `rider_earnings` y `collect_amount`

#### Scenario: Rider recibe order_accepted con pago bancario
- **WHEN** el rider acepta exitosamente una orden con pago bancario
- **THEN** el mensaje `order_accepted` incluye `pickup_code` y `rider_earnings`, pero NO `collect_amount`
