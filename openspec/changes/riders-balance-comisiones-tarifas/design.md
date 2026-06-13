## Context

El sistema ya tiene el rol `Rider` en `internal/auth/domain/models/role.go` y el flujo de confirmación de entrega (`ConfirmDelivery`) en `internal/orders/app/order.go`. Lo que falta es el módulo financiero que cierra el ciclo: al completar un pedido, calcular el monto de la carrera y ajustar el balance del rider.

No existe actualmente un módulo `riders` en `internal/`. La implementación requerirá crear uno siguiendo la misma estructura hexagonal del resto del proyecto.

## Goals / Non-Goals

**Goals:**

- Módulo `riders` con balance, movimientos, umbrales y retiro.
- Cálculo automático disparado desde `orders.ConfirmDelivery`.
- Endpoints `GET /balance`, `GET /transactions`, `POST /withdrawal`.
- Restricción de asignación según umbrales (integrable en el flujo de asignación futuro).

**Non-Goals:**

- Asignación automática de carreras a riders (fuera del alcance de esta feature).
- Notificaciones push al rider (puede agregarse como feature separada).
- Conversión de moneda (el balance se maneja en USD).
- Procesamiento del retiro hacia una cuenta bancaria (solo se registra la solicitud).

## Decisions

### 1. Nuevo módulo `internal/riders/` con arquitectura hexagonal completa

El balance del rider es un bounded context distinto de `orders`. El módulo tendrá:

```
internal/riders/
  domain/
    models/    — Rider, BalanceMovement
    ports/     — RidersApp, RidersRepo
    dtos/      — BalanceResponse, TransactionResponse, WithdrawalResponse
  app/         — lógica de balance, tarifa y retiro
  repo/        — queries a riders.balance_movements + perfil rider
  handler/     — HTTP handlers
```

**Alternativa descartada**: Agregar el balance directamente en el módulo `orders`. Descartada porque el balance es una preocupación del rider, no del pedido; mezclarlos viola la separación de bounded contexts.

### 2. Integración con `orders` vía interfaz de puerto (no llamada directa)

`orders/app/order.go` NO importará el módulo `riders` directamente. En su lugar:

- Se define `ports.RiderBalanceUpdater` en `internal/orders/domain/ports/`:
  ```go
  type RiderBalanceUpdater interface {
      RecordDelivery(ctx context.Context, orderID, riderID uuid.UUID, paymentMethod string, distanceKm float64, createdAt time.Time) error
  }
  ```
- `orderApp` recibe esta interfaz como dependencia opcional (inyectada por DI).
- Si `RiderBalanceUpdater` es nil o falla, `ConfirmDelivery` completa igual (ver spec `order-completion`).

**Alternativa descartada**: Evento Redis Stream en `ConfirmDelivery`. Añade latencia y complejidad innecesaria para un cálculo síncrono simple que no necesita distribución.

### 3. Schema de BD en `riders` schema de PostgreSQL

Mantener separación de schemas de PostgreSQL:

```sql
-- Nuevo campo en tabla del rider (o tabla riders si no existe)
ALTER TABLE riders.riders ADD COLUMN balance NUMERIC(10,2) NOT NULL DEFAULT 0;

-- Nueva tabla de movimientos
CREATE TABLE riders.balance_movements (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id      UUID NOT NULL REFERENCES riders.riders(id),
    order_id      UUID NOT NULL,
    amount        NUMERIC(10,2) NOT NULL,  -- monto_carrera bruto
    net_amount    NUMERIC(10,2) NOT NULL,  -- ganancia rider (80%)
    commission    NUMERIC(10,2) NOT NULL,  -- comisión plataforma (20%)
    tariff_rate   NUMERIC(5,3) NOT NULL,   -- 0.40 o 0.60
    distance_km   NUMERIC(8,3) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 4. Tarifas como constantes en el dominio

Las tarifas diurna/nocturna son reglas de negocio del dominio, no configuración de DB. Se definen como constantes en `internal/riders/app/`:

```go
const (
    TariffDayRate   = 0.40  // antes de las 22:00
    TariffNightRate = 0.60  // desde las 22:00
    PlatformCommissionRate = 0.20
    BalanceRiskThreshold   = -15.0
    BalanceNormalThreshold = -5.0
    BalanceWithdrawalMin   = 15.0
)
```

**Alternativa descartada**: Guardar tarifas en DB para hacerlas configurables. No hay requisito de configurabilidad en esta etapa; la complejidad no está justificada.

## Risks / Trade-offs

- **[Riesgo] `distance_km` no existe en el modelo actual de `Order`** → Si el campo no está en la tabla, los movimientos no se pueden calcular. **Mitigación**: Verificar si `orders.orders` tiene `distance_km`; si no, agregar como parte de la migración o como prerequisito bloqueante.

- **[Trade-off] Integración síncrona en `ConfirmDelivery`** → Añade latencia mínima a la confirmación de entrega. A cambio, el balance es inmediatamente consistente. Aceptable dado el volumen esperado.

- **[Riesgo] Doble aplicación** → Si `ConfirmDelivery` se llama dos veces por algún bug, el balance se ajustaría dos veces. **Mitigación**: La constraint `delivery_code_used_at NOT NULL` ya previene esto; el código falla antes de llegar al ajuste de balance.

## Migration Plan

1. Verificar existencia de `distance_km` en `orders.orders`.
2. Migración `000NN_rider_balance.sql`:
   - Crear tabla `riders.balance_movements`.
   - Agregar columna `balance` al perfil del rider.
3. Deploy sin datos a migrar (balance inicial = 0 para todos los riders).
4. Rollback: `-- +goose Down` con `DROP TABLE` y `DROP COLUMN`.

## Open Questions

- ¿Existe ya una tabla `riders.riders`? ¿O el perfil del rider está en `users.users` con un campo de rol? Determinar esto antes de escribir la migración.
- ¿El campo `distance_km` existe en `orders.orders`? Si no, ¿quién lo calcula y cuándo?
- ¿Hay un mecanismo de asignación de riders a pedidos que deba integrarse con los umbrales? Si no existe aún, el control de asignación puede implementarse como validación en el endpoint futuro.
