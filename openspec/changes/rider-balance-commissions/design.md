## Context

Actualmente el dominio `riders` solo gestiona la asignación de órdenes vía WebSocket. No existe ningún registro financiero para los riders: ni balance, ni historial de movimientos, ni control de riesgo por tipo de pago. En el futuro los shops también necesitarán balance, por lo que el modelo financiero debe ser genérico desde el inicio.

La arquitectura es hexagonal (Ports & Adapters) con bun ORM, goose migrations, uber/dig para DI y Echo v4 para HTTP. Ya existe lógica de distancia con Google Maps Distance Matrix API en `internal/shops/repo/shops.go` (`calculateDistances`) que será reutilizada como referencia.

## Goals / Non-Goals

**Goals:**
- Introducir el schema `finance` con tablas `balances` y `transactions` discriminadas por `entity_type`, reutilizables para shops y otros actores futuros.
- Calcular automáticamente el monto de carrera al completar una orden usando Google Maps Distance Matrix (shop → dirección del cliente).
- Actualizar el balance del rider según el tipo de pago al completar cada carrera.
- Filtrar asignación de carreras por tipo de pago cuando el rider está en zona de riesgo.
- Exponer tres endpoints REST para que el rider consulte su balance, transacciones y solicite retiro.
- Enriquecer los mensajes WebSocket con datos de ganancia.

**Non-Goals:**
- Balance de shops (se habilitará en un cambio futuro; la infra queda lista aquí).
- Procesamiento real de retiros (pagos bancarios, transferencias): este cambio solo registra la solicitud.
- Ocultar órdenes de efectivo a riders en zona de riesgo en el WebSocket (feature futura separada; por ahora solo se valida al aceptar).
- Dashboard de comisiones de la plataforma.

## Decisions

### D1 — Schema `finance` compartido con `entity_type`

**Decisión:** una tabla `finance.balances(entity_id UUID, entity_type TEXT, amount NUMERIC)` y `finance.transactions(id, entity_id, entity_type, order_id, amount, movement_type, payment_method, created_at)`.

**Alternativas consideradas:**
- *Tablas por dominio* (`riders.balance`, `shops.balance`): duplica estructura y migraciones cuando lleguen los shops.
- *JSON en el perfil del rider*: no es consultable ni trazable.

**Rationale:** el patrón de ledger genérico es estándar en fintech. El overhead de filtrar por `entity_type` es despreciable frente al beneficio de no duplicar infraestructura cuando se agregue el balance de shops.

---

### D2 — Cálculo de monto de carrera en el dominio `orders` al completar

**Decisión:** al transicionar una orden a estado `Completed` (en `internal/orders/app/order.go`), se llama a un nuevo port `FinancePort.RecordRideEarning(ctx, order)`. La lógica de tarifa y comisión vive en el dominio `riders`, implementada en su capa `app`.

**Alternativas consideradas:**
- *Calcular en `riders` vía evento Redis*: introduce latencia y complejidad de eventual consistency innecesaria para este caso.
- *Calcular directamente en `orders`*: viola la separación de bounded contexts.

**Rationale:** inyectar un port en `orders` mantiene los bounded contexts separados sin mensajería asíncrona. La dependencia es unidireccional: `orders` → `FinancePort`.

---

### D3 — Distancia calculada con Google Maps Distance Matrix (shop → cliente)

**Decisión:** reutilizar el patrón de `internal/shops/repo/shops.go` (`calculateDistances`). La distancia de la carrera es la ruta en coche desde las coordenadas del shop hasta la dirección de entrega del cliente (`order.Latitude`, `order.Longitude`). Se usa `config.Get().Google.Matrix` como API key.

**Rationale:** ya existe la integración y la API key en el proyecto. No tiene sentido introducir una aproximación euclidiana cuando la solución real ya está disponible.

---

### D4 — Umbral de riesgo verificado en `AcceptOrder`

**Decisión:** el filtro de zona de riesgo se aplica en `riders/app.AcceptOrder`: si el rider intenta aceptar una orden con pago en efectivo y su balance ≤ −$15, se retorna `ErrRiderInRiskZone` y el WebSocket responde `{"type":"error","message":"solo puedes aceptar carreras con pago bancario"}`.

**Nota:** en una feature futura se filtrará el broadcast para que los riders en zona de riesgo directamente no reciban órdenes de efectivo en el WebSocket. Esta validación en `AcceptOrder` permanecerá como segunda línea de defensa.

---

### D5 — Retiro registrado como transacción tipo `withdrawal`

**Decisión:** `POST /riders/withdrawal` crea una transacción con `movement_type = 'withdrawal'` y `amount` negativo, reduciendo el balance. No hay integración bancaria en este cambio.

## Risks / Trade-offs

- **[Riesgo] Race condition en `AcceptOrder` con verificación de balance** → Mitigation: la actualización de balance usa una transacción SQL con `SELECT FOR UPDATE` sobre `finance.balances`.
- **[Riesgo] Fallo en Google Maps al completar orden** → Mitigation: si la API falla, loguear el error y no bloquear la transición de estado; registrar el movimiento con `distance_km = 0` y marcar como `pending_recalculation` para reproceso manual.
- **[Trade-off] El rider ve órdenes de efectivo aunque esté en zona de riesgo** → Aceptado: se resuelve en feature futura que filtrará el broadcast. La validación en `AcceptOrder` previene la asignación incorrecta.

## Migration Plan

1. Ejecutar migración goose que crea schema `finance` + tablas `balances` y `transactions`.
2. Los riders existentes arrancan con `balance = 0` (fila se crea lazy en primer movimiento o al primer `GET /riders/balance`).
3. No hay rollback destructivo: el schema `finance` es aditivo y no toca tablas existentes.

## Open Questions

- ¿El retiro notifica a alguien (Discord, Slack) cuando se procesa? → Definir en el cambio de integración bancaria.
- ¿Qué coordenadas del shop se usan como origen? → Las coordenadas almacenadas en el shop (ya disponibles en el modelo `Shop`). Requiere hacer un query al repo de shops al completar la orden.
