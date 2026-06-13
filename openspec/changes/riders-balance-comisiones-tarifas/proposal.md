## Why

Los riders no tienen visibilidad de cuánto dinero de la plataforma tienen en su poder ni cuánto les debe la plataforma. Sin un sistema de balance, no es posible controlar el riesgo de que un rider acumule efectivo de la empresa indefinidamente ni habilitar retiros justos. La incorporación del rol `Rider` al sistema hace necesario cerrar este ciclo financiero.

## What Changes

- Nuevo módulo `riders` (bounded context) con modelo de balance y tabla de movimientos.
- Cálculo automático del monto de carrera al completar un pedido: `distancia_km × tarifa_horaria`.
- Ajuste de balance por tipo de pago al completar cada carrera (efectivo resta, bancario suma).
- Comisión de plataforma del 20% retenida por carrera; el rider recibe el 80%.
- Control de asignación de carreras según umbrales de balance del rider.
- Endpoints para consultar balance, historial de transacciones y solicitar retiro.

## Capabilities

### New Capabilities

- `rider-balance`: Registro y cálculo acumulado del balance del rider por tipo de pago (efectivo/bancario). Incluye historial de movimientos por carrera.
- `rider-km-tariffs`: Cálculo del monto de carrera según distancia en km y tarifa horaria (diurna $0.40/km antes de las 10 PM, nocturna $0.60/km desde las 10 PM). La tarifa se determina por el timestamp de creación del pedido.
- `rider-assignment-control`: Restricción de asignación de carreras según balance: riders con balance ≤ -$15 solo reciben carreras bancarias hasta que el balance supere -$5.
- `rider-withdrawal`: Solicitud manual de retiro cuando el balance es ≥ +$15. Descuenta el saldo disponible al procesar.

### Modified Capabilities

- `order-completion`: Al transicionar un pedido a `Completed`, se activa el cálculo del monto de carrera y el ajuste de balance del rider asignado.

## Impact

- Nuevo módulo `internal/riders/` con las capas hexagonales completas.
- Nueva migración de BD: tabla `riders.balance_movements` + campo `balance` en el perfil del rider.
- `internal/orders/app/order.go`: extender `ConfirmDelivery` (o el hook de `Completed`) para disparar el cálculo de balance.
- Nuevos endpoints bajo `/api/v1/riders/`.
- Nuevos errores sentinel en `internal/shared/errcustom/`.
- Nuevos providers en `cmd/server/providers/di.go`.
