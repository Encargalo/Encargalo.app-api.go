## ADDED Requirements

### Requirement: Schema finance con tablas de balance y transacciones
El sistema SHALL mantener un schema PostgreSQL `finance` con dos tablas genéricas discriminadas por `entity_type` para soportar múltiples actores financieros (riders, shops, etc.) sin duplicar estructura.

Tabla `finance.balances`:
- `entity_id UUID NOT NULL`
- `entity_type TEXT NOT NULL` (valores: `'rider'`, `'shop'`)
- `amount NUMERIC(12,2) NOT NULL DEFAULT 0`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- PK compuesta: `(entity_id, entity_type)`

Tabla `finance.transactions`:
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `entity_id UUID NOT NULL`
- `entity_type TEXT NOT NULL`
- `order_id UUID` (nullable, referencia lógica a la orden asociada)
- `amount NUMERIC(12,2) NOT NULL`
- `movement_type TEXT NOT NULL` (valores: `'ride_cash'`, `'ride_bank'`, `'withdrawal'`)
- `payment_method TEXT` (nullable, valores: `'cash'`, `'bank'`)
- `distance_km NUMERIC(8,3)`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

#### Scenario: Migración crea schema y tablas
- **WHEN** se ejecuta `goose up` con la migración del schema `finance`
- **THEN** el schema `finance` existe en la base de datos con las tablas `balances` y `transactions` con las columnas y constraints definidos

#### Scenario: Balances se crea lazy al primer movimiento
- **WHEN** se registra el primer movimiento financiero de un rider y no existe fila en `finance.balances` para ese `entity_id`
- **THEN** el sistema SHALL hacer upsert creando la fila con el monto inicial del movimiento

#### Scenario: Diferentes entity_type coexisten en la misma tabla
- **WHEN** existen registros con `entity_type = 'rider'` y `entity_type = 'shop'` en `finance.balances`
- **THEN** las consultas filtradas por `entity_type` retornan únicamente los registros del tipo solicitado
