## Why

Los riders no tienen un sistema financiero propio: no existe registro de cuánto efectivo tienen en su poder ni cuánto les debe la plataforma, lo que impide controlar el riesgo de asignación y dificulta los pagos entre partes. Este cambio introduce el balance del rider como entidad financiera central, diseñado desde el inicio como un sistema genérico que soporte en el futuro otros actores (shops, empleados, etc.) sin cambios estructurales.

## What Changes

- **Nuevo schema `finance`** con tablas compartidas `balances` y `transactions`, discriminadas por `entity_type` ('rider', 'shop', ...). Extensible a cualquier actor financiero futuro.
- **Tarifa dinámica por km**: diurna ($0.40/km) o nocturna ($0.60/km) según la hora de creación del pedido.
- **Comisión del 20%**: la plataforma retiene el 20% de cada carrera; el rider recibe el 80%.
- **Umbrales de asignación**: si el balance ≤ −$15 (zona de riesgo), el rider solo recibe carreras con pago bancario; se normaliza cuando balance > −$5.
- **Retiro manual**: el rider puede solicitar retiro cuando balance ≥ +$15; el saldo se descuenta al procesar.
- **Actualización del WebSocket de riders**: los mensajes `new_order` y `order_accepted` incluyen `rider_earnings` (ganancia neta) y `collect_amount` (monto a cobrar al cliente, solo en efectivo).
- **Tres endpoints nuevos** bajo `/riders/`:
  - `GET /riders/balance` — balance actual + últimos movimientos
  - `POST /riders/withdrawal` — solicitar retiro
  - `GET /riders/transactions` — historial paginado

## Capabilities

### New Capabilities

- `finance-schema`: Schema compartido `finance` con tablas `balances(entity_id, entity_type, amount)` y `transactions(entity_id, entity_type, order_id, amount, type, created_at)`. Base para todos los actores financieros de la plataforma.
- `rider-balance`: Balance acumulado del rider, cálculo de tarifa por km (diurna/nocturna), comisión del 20%, y actualización automática al completar una carrera.
- `rider-withdrawal`: Política de retiro manual con umbral mínimo de +$15 y descuento de saldo al procesar.
- `rider-transactions`: Historial de movimientos paginado, trazable por carrera.
- `rider-assignment-control`: Filtrado de asignación de carreras según el balance del rider (zona de riesgo vs. zona normal).

### Modified Capabilities

- `rider-websocket`: Los mensajes `new_order` y `order_accepted` agregan `rider_earnings` y, cuando el pago es en efectivo, `collect_amount`.

## Impact

- **Base de datos**: nuevo schema `finance` con 2 tablas + migración goose. No hay tablas por dominio separadas.
- **`internal/riders/`**: nuevos ports, app layer y repo para balance, retiro y transacciones (consultan `finance.*` con `entity_type = 'rider'`).
- **`internal/orders/app/`**: al transicionar una orden a `Completed` se dispara el cálculo de tarifa y el ajuste de balance en `finance`.
- **`internal/riders/handler/riders.go`**: mensajes WebSocket actualizados con datos de ganancia.
- **`internal/riders/app/riders.go`**: lógica de asignación extendida con verificación de balance antes de broadcast.
- **Router**: 3 rutas nuevas bajo `/riders/` con rol `Rider`.
- **Swagger + Postman**: documentación de los 3 endpoints nuevos.
