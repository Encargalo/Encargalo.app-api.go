# Plan técnico — Issue #184: unificar validación de pagos, balance y reembolsos entre orders y trips

> Implementado en el working tree. Este archivo queda como registro de las decisiones; el código vive en `internal/payments` y en los módulos que se enchufan (`orders`, `users`, `finance`). El subject de trips sigue fuera de alcance (issue #183).

## 1. Contexto y motivación

Hoy `internal/orders/app/payment_reviews.go` valida comprobantes de pago (Pago Móvil), acredita/refunda balance del customer y muta el estado de la orden — todo acoplado a `models.Order` y a un campo `OrderID` fijo en `PaymentReview`/`Transaction`/`BalanceRefundRequest`.

Issue #183 (trips) necesita el mismo flujo de validación de comprobante, pero **no** debe duplicarse un `payment_reviews.go` por dominio. Se generaliza a un módulo compartido `internal/payments`, con `ReferenceID uuid.UUID + ReferenceType string` en vez de `OrderID`.

**Hallazgo adicional (confirmado por investigación, incluido en alcance por decisión explícita del usuario):** `internal/finance/domain/models.Transaction` (usado por riders/shops, mapeado a la MISMA tabla física `finance.transactions` que `internal/users.Transaction`) tiene el mismo bug de raíz — `internal/riders/app/finance.go:42` ya reusa `OrderID` también para IDs de trip, sin discriminar. Se generaliza también ese lado en este mismo PR.

**Motivación de negocio:** Staff necesita ver órdenes y viajes pendientes de validar pago en **una sola cola** (un solo WS), sin tener que saber manualmente la lógica de negocio de cada dominio — el server decide qué hacer según `ReferenceType`.

Dependencias: #133 (implementación original de referencia en orders, que este issue generaliza). Relacionado: #182 (precio fijo de trip — ya implementado), #183 (trips consumirá este módulo una vez unificado, no se implementa acá).

## 2. Decisiones ya acordadas con el usuario

1. **Alcance ampliado**: incluir la generalización de `internal/finance` (riders/shops) en este mismo PR, no dejarlo para una issue aparte — mismo root cause que `internal/users`, misma tabla física.
2. **Migración de `orders.payment_reviews`**: se **recrea** (DROP + CREATE) en un schema nuevo `payments` — los datos existentes no importan (no hay operación real en producción todavía). No hace falta backfill cuidadoso ni preservar histórico real.
3. **WS de Staff**: cambio de path **directo, sin alias de compatibilidad** — `GET /orders/staff/reviews` se reemplaza por `GET /payments/staff/reviews`. Esto rompe al CRM (`Encargalo.app-CRM`, repo aparte) — coordinar el aviso queda **fuera** de este PR (es tarea de equipo/comunicación, no de código).
4. **Fuera de alcance explícito**: implementar el lado de `trips` (`PaymentSubject` para `ReferenceType="trip"`) es la issue #183, futura. Este PR debe dejar el registro/dispatch **listo** para que trips se enchufe sin tocar `payments/app`, pero no se implementa el subject de trips acá.
5. **`internal/riders/`**: sin cambios. `riders/domain/models.Transaction` sigue usando columnas dedicadas `order_id`+`trip_id` (con FK real a `trips.trips`) — un patrón distinto y deliberadamente no tocado en este PR.

## 3. Puntos de diseño que quedaron abiertos tras la investigación, y su resolución

Estos 4 puntos surgieron al bajar el diseño a código concreto y no estaban decididos de antemano — se resuelven acá con su justificación, para que la implementación no tenga que re-decidirlos:

### 3.1 ¿Cómo resuelve `payments/app` el `customerID` para acreditar balance?

`ApproveOverpay`/`RejectUnderpaid` necesitan saber a qué customer acreditarle Bs, pero `PaymentSubject` (4 métodos: `ExpectedAmountBs`, `OnApproved`, `OnOverpaid`, `OnRejected`) no lo expone — solo `orders`/`trips` saben quién es el dueño de su propia entidad.

**Resolución**: agregar un 5º método de solo lectura a `PaymentSubject`: `CustomerID(ctx, referenceID) (uuid.UUID, error)`. No contradice que el crédito de balance viva centralizado en `payments/app` — solo le da el dato que falta para hacerlo.

### 3.2 ¿Cómo recupera `orderPaymentSubject` el detalle del `PaymentReview` ya resuelto?

`syncPaymentDetails` (hoy en `orders/app/payment_reviews.go`) escribe el estado del review dentro de `order.Details` (JSON que el customer ve vía WS `order_update`) — necesita el `PaymentReview` completo (status final, `CreditedBs`, `DifferenceBs`, `RejectionReason`), pero `PaymentSubject.On*` solo recibe `referenceID` (+ `diffBs`/`reason` sueltos).

**Resolución**: `orderPaymentSubject` recibe también `paymentsPorts.PaymentReviewsRepo` como dependencia, y hace `GetByID(ctx, referenceID)` dentro de cada `On*` para releer el review ya resuelto (post-CAS). Una lectura extra a `payments.reviews`, sin impacto real de performance (es un flujo de Staff, no un hot path).

### 3.3 ¿Cómo no perder el enriquecimiento (`shop_id`, `customer_name`, `total_usd`) en el snapshot/broadcast de Staff?

`payments/app` no puede conocer `Order` — pero si `ListActiveForStaff`/el broadcast solo devuelven los campos genéricos, Staff pierde contexto que hoy sí ve (a qué shop pertenece, nombre del customer, monto total de la orden). Sería una regresión funcional real, no cosmética.

**Resolución**: agregar un segundo punto de extensión por DI, simétrico al `PaymentSubjectRegistry` pero para enriquecimiento de lectura: `StaffReviewHub.SetSnapshotProvider(fn func(ctx) ([]models.PaymentReviewBroadcast, error))`. `orders/app` registra su propio snapshot provider (que sí puede hacer join con `orders.orders`) en `cmd/server/main.go`, mismo patrón que los setters post-construcción ya usados hoy (`hub.SetReviewsApp`, `refundHub.SetFinanceApp`).

### 3.4 ¿Se elimina la columna `finance.transactions.order_id`?

La decisión original (#2 de arriba, sección de decisiones) sugería "recomiendo eliminarla" — **esto se corrige** tras confirmar que `internal/riders/domain/models.Transaction` (fuera de alcance, no se toca) sigue usando esa columna con FK real para las filas `entity_type='rider'`.

**Resolución**: `finance.transactions.order_id` **NO se elimina**. Se agregan `reference_id`/`reference_type` como columnas nuevas junto a `order_id`. Los structs Go que sí se generalizan (`internal/users.Transaction`, `internal/finance.Transaction` del lado customer/staff-payout) simplemente dejan de mapear `order_id` — bun permite mapeo parcial de columnas sin problema.

## 4. Arquitectura — módulo nuevo `internal/payments`

Mismo layout hexagonal que el resto del repo:

```
internal/payments/
  domain/
    models/
      payment_review.go
      payment_review_broadcast.go
    dtos/
      staff_ws.go
    ports/
      payment_reviews.go
      finance.go
      subject.go
  app/
    payment_reviews.go
    subject_registry.go
    staff_review_hub.go
  repo/
    payment_reviews.go
  handler/
    staff_reviews.go
```

### 4.1 `internal/payments/domain/models/payment_review.go`

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PaymentReviewStatus string

const (
	ReviewPendingReview      PaymentReviewStatus = "pending_review"
	ReviewApproved           PaymentReviewStatus = "approved"
	ReviewOverpaid           PaymentReviewStatus = "overpaid"
	ReviewRejectedUnderpaid  PaymentReviewStatus = "rejected_underpaid"
	ReviewRejectedInvalid    PaymentReviewStatus = "rejected_invalid"
	ReviewRefundedShopReject PaymentReviewStatus = "refunded_shop_reject"
)

// ReferenceTypeOrder es el único tipo registrado por este PR. Trips (#183)
// agrega su propia constante (ej. ReferenceTypeTrip = "trip") en su propio
// paquete al implementar PaymentSubject — este archivo no necesita tocarse.
const ReferenceTypeOrder = "order"

const PaymentProofAlertInterval = 3 * time.Minute

type PaymentReview struct {
	bun.BaseModel `bun:"table:payments.reviews"`

	ID               uuid.UUID           `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ReferenceID      uuid.UUID           `bun:"reference_id,type:uuid,notnull" json:"reference_id"`
	ReferenceType    string              `bun:"reference_type,notnull" json:"reference_type"`
	ProofURL         string              `bun:"proof_url,notnull" json:"proof_url"`
	ExpectedAmountBs float64             `bun:"expected_amount_bs,notnull" json:"expected_amount_bs"`
	ObservedAmountBs *float64            `bun:"observed_amount_bs" json:"observed_amount_bs,omitempty"`
	DifferenceBs     *float64            `bun:"difference_bs" json:"difference_bs,omitempty"`
	BcvRate          float64             `bun:"bcv_rate,notnull" json:"bcv_rate"`
	Status           PaymentReviewStatus `bun:"status,notnull" json:"status"`
	RejectionReason  *string             `bun:"rejection_reason" json:"rejection_reason,omitempty"`
	StaffID          *uuid.UUID          `bun:"staff_id,type:uuid" json:"staff_id,omitempty"`
	CreditedBs       *float64            `bun:"credited_bs" json:"credited_bs,omitempty"`
	ReferenceLast4   *string             `bun:"reference_last4" json:"reference_last4,omitempty"`
	CreatedAt        time.Time           `bun:"created_at,notnull,default:now()" json:"created_at"`
	ResolvedAt       *time.Time          `bun:"resolved_at" json:"resolved_at,omitempty"`
	LastAlertedAt    *time.Time          `bun:"last_alerted_at,nullzero" json:"last_alerted_at,omitempty"`
}

// StaffReviewResult es genérico: payments/app no conoce el shape de Order/Trip.
type StaffReviewResult struct {
	ReviewID        uuid.UUID `json:"review_id"`
	ReferenceID     uuid.UUID `json:"reference_id"`
	ReferenceType   string    `json:"reference_type"`
	Action          string    `json:"action"`
	DuplicateReject bool      `json:"duplicate_reject,omitempty"`
	RejectionReason string    `json:"rejection_reason,omitempty"`
}
```

### 4.2 `internal/payments/domain/models/payment_review_broadcast.go`

Mismo shape que `orders/domain/models/payment_review_broadcast.go` hoy, renombrando `OrderID→ReferenceID` y agregando `ReferenceType`. Los campos "order-shaped" (`ShopID`, `CustomerID`, `CustomerName`, `CustomerPhone`, `BalanceAmountBs`, `PagoMovilAmountBs`, `TotalUSD`, `OrderStatus`) se preservan tal cual — documentados como poblados solo cuando `ReferenceType=="order"`, hasta que trips (#183) decida cómo enriquecer los suyos.

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentReviewBroadcast struct {
	Type              string              `json:"type"`
	ReviewID          uuid.UUID           `json:"review_id"`
	ReferenceID       uuid.UUID           `json:"reference_id"`
	ReferenceType     string              `json:"reference_type"`
	ShopID            uuid.UUID           `json:"shop_id,omitempty"`
	CustomerID        uuid.UUID           `json:"customer_id,omitempty"`
	CustomerName      string              `json:"customer_name,omitempty"`
	CustomerPhone     string              `json:"customer_phone,omitempty"`
	ProofURL          string              `json:"proof_url"`
	ExpectedAmountBs  float64             `json:"expected_amount_bs"`
	ObservedAmountBs  *float64            `json:"observed_amount_bs,omitempty"`
	DifferenceBs      *float64            `json:"difference_bs,omitempty"`
	Status            PaymentReviewStatus `json:"status"`
	RejectionReason   *string             `json:"rejection_reason,omitempty"`
	CreditedBs        *float64            `json:"credited_bs,omitempty"`
	BalanceAmountBs   float64             `json:"balance_amount_bs,omitempty"`
	PagoMovilAmountBs float64             `json:"pago_movil_amount_bs,omitempty"`
	TotalUSD          float64             `json:"total_usd,omitempty"`
	OrderStatus       string              `json:"order_status,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
}
```

### 4.3 `internal/payments/domain/dtos/staff_ws.go`

```go
package dtos

import "github.com/google/uuid"

// StaffWSMessage: mensajes entrantes desde el WS de staff. Mismo shape que
// hoy usa orders/handler/staffReviews.go — sin cambios de campos.
type StaffWSMessage struct {
	Type             string    `json:"type"`
	ReviewID         uuid.UUID `json:"review_id"`
	ObservedAmountBs *float64  `json:"observed_amount_bs,omitempty"`
	ReferenceLast4   string    `json:"reference_last4,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}
```

`action_ok`/`error`/`connected` se siguen serializando como `map[string]any` en el handler (mismo patrón que hoy) — no hace falta struct dedicado.

### 4.4 `internal/payments/domain/ports/payment_reviews.go`

```go
package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/payments/domain/models"
	"github.com/google/uuid"
)

type PaymentReviewsRepo interface {
	Create(ctx context.Context, review *models.PaymentReview) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PaymentReview, error)
	ListActiveForStaff(ctx context.Context) ([]models.PaymentReview, error)
	ResolvePending(ctx context.Context, id uuid.UUID, update *models.PaymentReview) (bool, error)
	ExistsResolvedReferenceLast4(ctx context.Context, last4 string) (bool, error)
	// ListPendingForAlert filtra opcionalmente por referenceType ("" = todos).
	ListPendingForAlert(ctx context.Context, referenceType string, alertBefore time.Time) ([]models.PaymentReview, error)
	MarkAlerted(ctx context.Context, id uuid.UUID, alertedAt, alertBefore time.Time) (bool, error)
}

type ManagerConn interface {
	Send(msg []byte)
	Close()
}

type StaffReviewHub interface {
	Register(ctx context.Context, staffID uuid.UUID, conn ManagerConn) error
	Unregister(staffID uuid.UUID, conn ManagerConn)
	BroadcastReview(review models.PaymentReviewBroadcast) error
	// SetSnapshotProvider registra cómo construir el snapshot inicial (al
	// conectar) enriquecido — orders/app lo setea en main.go con su propio
	// join a Order. Ver sección 3.3.
	SetSnapshotProvider(fn func(ctx context.Context) ([]models.PaymentReviewBroadcast, error))
}
```

### 4.5 `internal/payments/domain/ports/finance.go`

```go
package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CustomerBalanceCreditor: idéntico al de orders hoy (ordersPorts.CustomerBalanceCreditor),
// generalizado a referenceID/referenceType. Implementado por
// users/app.customerFinanceApp, adaptado vía DI igual que hoy.
type CustomerBalanceCreditor interface {
	GetBalanceBs(ctx context.Context, customerID uuid.UUID) (float64, error)
	GetBalancePaymentDebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string) (float64, error)
	CreditPaymentOverage(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditPaymentPartial(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditOrderRejected(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, refundablePMBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	DebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error
	RefundBalancePayment(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error
}
```

`CreditOrderRejected` conserva el nombre (el `MovementType` real es `order_rejected_refund`, vocabulario propio de orders) — no se renombra genéricamente para minimizar el diff en `users/domain/models.MovementType`; trips definirá su propio nombre semántico cuando se implemente.

### 4.6 `internal/payments/domain/ports/subject.go`

```go
package ports

import (
	"context"

	"github.com/google/uuid"
)

// PaymentSubject es implementado por cada dominio que puede tener reviews de
// pago (orders hoy; trips en la issue #183). payments/app resuelve el CAS,
// detecta duplicados y acredita balance de forma genérica, y delega en el
// PaymentSubject registrado para reference_type la mutación específica del
// dominio (mover el Order/Trip de estado, liberar al shop, etc.).
type PaymentSubject interface {
	ExpectedAmountBs(ctx context.Context, referenceID uuid.UUID) (float64, error)
	CustomerID(ctx context.Context, referenceID uuid.UUID) (uuid.UUID, error)
	OnApproved(ctx context.Context, referenceID uuid.UUID) error
	OnOverpaid(ctx context.Context, referenceID uuid.UUID, diffBs float64) error
	OnRejected(ctx context.Context, referenceID uuid.UUID, reason string) error
}

// PaymentSubjectRegistry resuelve el PaymentSubject correcto por referenceType.
// Se llena por DI (cmd/server/main.go, setter post-construcción), nunca con
// un switch en payments/app. Trips solo necesita agregar una línea de
// Register — no toca payments/app.
type PaymentSubjectRegistry interface {
	Register(referenceType string, subject PaymentSubject)
	Get(referenceType string) (PaymentSubject, error)
}
```

### 4.7 `internal/payments/app/subject_registry.go`

```go
package app

import (
	"fmt"
	"sync"

	"Encargalo.app-api.go/internal/payments/domain/ports"
)

// SubjectRegistry (exportado: cmd/server/providers/di.go necesita referenciar
// el tipo concreto para el adaptador a la interfaz, mismo patrón que
// StaffReviewHubImpl).
type SubjectRegistry struct {
	mu       sync.RWMutex
	subjects map[string]ports.PaymentSubject
}

func NewSubjectRegistry() *SubjectRegistry {
	return &SubjectRegistry{subjects: make(map[string]ports.PaymentSubject)}
}

func (r *SubjectRegistry) Register(referenceType string, subject ports.PaymentSubject) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subjects[referenceType] = subject
}

func (r *SubjectRegistry) Get(referenceType string) (ports.PaymentSubject, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.subjects[referenceType]
	if !ok {
		return nil, fmt.Errorf("payments: no hay PaymentSubject registrado para reference_type %q", referenceType)
	}
	return s, nil
}

var _ ports.PaymentSubjectRegistry = (*SubjectRegistry)(nil)
```

### 4.8 `internal/payments/app/payment_reviews.go` — lo genérico

`paymentReviewsApp` depende de `repo ports.PaymentReviewsRepo`, `finance ports.CustomerBalanceCreditor`, `hub ports.StaffReviewHub`, `registry ports.PaymentSubjectRegistry`.

- **`CreateReview(ctx, referenceType string, referenceID uuid.UUID, proofURL string, expectedBs, bcvRate float64) (*models.PaymentReview, error)`**: valida + persiste. Disponible para cuando trips lo necesite (#183) — **no se conecta a ningún flujo real de orders en este PR** (orders sigue creando el review directamente vía `reviewsRepo.Create`, ver sección 6, para no arriesgar el flujo transaccional de `CreateOrder` que ya tiene rollbacks manuales delicados).
- **`Approve(ctx, reviewID, staffID uuid.UUID, referenceLast4 string) (models.StaffReviewResult, error)`**:
  1. Detección de duplicado (`ExistsResolvedReferenceLast4` — única GLOBAL a toda la tabla, sin importar `ReferenceType`, para evitar reusar un comprobante entre una orden y un viaje) → si duplicado, auto-rechaza llamando internamente a la misma lógica de `RejectInvalid`.
  2. `loadPending`: `repo.GetByID` + valida `review.Status == pending_review` (ya NO valida estado de Order/Trip — eso es responsabilidad de `PaymentSubject.OnApproved`, que debe devolver error si la referencia ya no está en el estado esperado).
  3. `repo.ResolvePending` (CAS).
  4. `registry.Get(review.ReferenceType)` → `subject.OnApproved(ctx, review.ReferenceID)`.
  5. `hub.BroadcastReview(...)` con el estado final (para que Staff vea el resultado sin refrescar).
- **`ApproveOverpay`**: igual + `observedBs > expected`, calcula `diff`, `subject.CustomerID(ctx, referenceID)` para resolver a quién, `finance.CreditPaymentOverage(ctx, customerID, referenceID, referenceType, creditedBs, bcvRate)`, `subject.OnOverpaid(ctx, referenceID, diff)`.
- **`RejectUnderpaid`**: `observedBs < expected`, `finance.CreditPaymentPartial(...)`, `subject.OnRejected(ctx, referenceID, "underpaid")`.
- **`RejectInvalid`**: requiere `reason`, sin crédito, `subject.OnRejected(ctx, referenceID, reason)`.
- **`ListActiveForStaff`**: delega al `SetSnapshotProvider` si está registrado (fallback a lista genérica sin enriquecer si no hay ninguno registrado para pruebas/dev).

### 4.9 `internal/payments/app/staff_review_hub.go`

Copia 1:1 de `orders/app/staff_review_hub.go`, moviendo el paquete y agregando el campo `snapshotProvider func(ctx) ([]models.PaymentReviewBroadcast, error)` + método `SetSnapshotProvider`.

### 4.10 `internal/payments/repo/payment_reviews.go`

Copia 1:1 de `orders/repo/payment_reviews.go`, apuntando a `payments.reviews`, `s/order_id/reference_id/`, filtro opcional `reference_type` en `ListPendingForAlert`.

### 4.11 `internal/payments/handler/staff_reviews.go`

Copia de `orders/handler/staffReviews.go`. `action_ok` cambia de shape:
```json
{"type":"action_ok","action":"approve","review_id":"...","reference_id":"...","reference_type":"order"}
```
(sin `order_id`/`order_status` — Staff puede leer el estado final del `payment_review_update` que ya se emite en la resolución, punto 4.8 paso 5). Swagger: `@Router /payments/staff/reviews [get]`, `@Tags Payments`.

## 5. Migraciones (goose, desde `00074` — última existente es `00073_accounts_normalize_empty_email.sql`)

### `00074_payments_schema.sql`

```sql
-- +goose Up
CREATE SCHEMA IF NOT EXISTS payments;

CREATE TABLE payments.reviews (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id        UUID          NOT NULL,
    reference_type      TEXT          NOT NULL DEFAULT 'order',
    proof_url           TEXT          NOT NULL,
    expected_amount_bs  NUMERIC(14,2) NOT NULL,
    observed_amount_bs  NUMERIC(14,2),
    difference_bs       NUMERIC(14,2),
    bcv_rate            NUMERIC(14,6) NOT NULL DEFAULT 0,
    status              TEXT          NOT NULL DEFAULT 'pending_review',
    rejection_reason    TEXT,
    staff_id            UUID REFERENCES users.accounts (id),
    credited_bs         NUMERIC(12,2),
    reference_last4     CHAR(4),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    last_alerted_at     TIMESTAMPTZ
);

CREATE INDEX idx_payments_reviews_status ON payments.reviews (status);
CREATE INDEX idx_payments_reviews_reference ON payments.reviews (reference_type, reference_id, created_at DESC);
CREATE INDEX idx_payments_reviews_alert ON payments.reviews (status, last_alerted_at) WHERE status = 'pending_review';
CREATE UNIQUE INDEX idx_payments_reviews_reference_last4_unique
    ON payments.reviews (reference_last4)
    WHERE reference_last4 IS NOT NULL AND status IN ('approved', 'overpaid', 'rejected_underpaid');

-- Datos existentes no importan (sin operación real todavía, confirmado) —
-- se recrea en vez de migrar/preservar histórico.
DROP TABLE IF EXISTS orders.payment_reviews;

-- +goose Down
CREATE TABLE IF NOT EXISTS orders.payment_reviews (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID          NOT NULL REFERENCES orders.orders (id),
    proof_url           TEXT          NOT NULL,
    expected_amount_bs  NUMERIC(14,2) NOT NULL,
    observed_amount_bs  NUMERIC(14,2),
    difference_bs       NUMERIC(14,2),
    bcv_rate            NUMERIC(14,6) NOT NULL DEFAULT 0,
    status              TEXT          NOT NULL DEFAULT 'pending_review',
    rejection_reason    TEXT,
    staff_id            UUID REFERENCES users.accounts (id),
    credited_bs         NUMERIC(12,2),
    reference_last4     CHAR(4),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    last_alerted_at     TIMESTAMPTZ
);
CREATE INDEX idx_payment_reviews_status ON orders.payment_reviews (status);
CREATE INDEX idx_payment_reviews_order ON orders.payment_reviews (order_id, created_at DESC);
CREATE UNIQUE INDEX idx_payment_reviews_reference_last4_unique
    ON orders.payment_reviews (reference_last4)
    WHERE reference_last4 IS NOT NULL AND status IN ('approved', 'overpaid', 'rejected_underpaid');
CREATE INDEX idx_payment_reviews_alert ON orders.payment_reviews (status, last_alerted_at) WHERE status = 'pending_review';

DROP SCHEMA IF EXISTS payments CASCADE;
```

### `00075_finance_transactions_reference.sql`

```sql
-- +goose Up
-- reference_id/reference_type generalizan order_id para customer (users) y
-- staff_payout (finance) transactions. La columna order_id NO se elimina:
-- sigue siendo usada por finance.transactions con entity_type='rider'
-- (RecordRideEarning/RecordTripEarning, internal/riders — fuera de alcance).
ALTER TABLE finance.transactions
    ADD COLUMN IF NOT EXISTS reference_id   UUID,
    ADD COLUMN IF NOT EXISTS reference_type TEXT;

CREATE INDEX IF NOT EXISTS idx_finance_transactions_reference
    ON finance.transactions (reference_type, reference_id);

-- +goose Down
DROP INDEX IF EXISTS finance.idx_finance_transactions_reference;
ALTER TABLE finance.transactions
    DROP COLUMN IF EXISTS reference_type,
    DROP COLUMN IF EXISTS reference_id;
```

### `00076_balance_refund_requests_reference.sql`

```sql
-- +goose Up
-- balance_refund_requests.order_id solo lo usa internal/users (sin riesgo
-- de romper otro módulo) -> se puede renombrar directo.
ALTER TABLE finance.balance_refund_requests
    ADD COLUMN IF NOT EXISTS reference_type TEXT NOT NULL DEFAULT 'order';

ALTER TABLE finance.balance_refund_requests
    RENAME COLUMN order_id TO reference_id;

-- +goose Down
ALTER TABLE finance.balance_refund_requests
    RENAME COLUMN reference_id TO order_id;
ALTER TABLE finance.balance_refund_requests
    DROP COLUMN IF EXISTS reference_type;
```

## 6. Archivos existentes a modificar

### `internal/orders/`

- **Eliminar**: `domain/models/payment_review.go`, `payment_review_broadcast.go`, `domain/ports/payment_reviews.go` (contenido migrado a `payments`), `repo/payment_reviews.go`, `handler/staffReviews.go`, `app/staff_review_hub.go`.
- **Nuevo `internal/orders/app/payment_subject.go`** (reemplaza a `payment_reviews.go` actual): implementa `paymentsPorts.PaymentSubject` para `ReferenceType="order"`.

```go
package app

type orderPaymentSubject struct {
	orders  ports.OrdersRepo
	reviews paymentsPorts.PaymentReviewsRepo // para releer el review resuelto (syncPaymentDetails, sección 3.2)
	mgr     ports.AppManagerOrder
}

func NewOrderPaymentSubject(orders ports.OrdersRepo, reviews paymentsPorts.PaymentReviewsRepo, mgr ports.AppManagerOrder) paymentsPorts.PaymentSubject {
	return &orderPaymentSubject{orders, reviews, mgr}
}

func (s *orderPaymentSubject) ExpectedAmountBs(ctx context.Context, referenceID uuid.UUID) (float64, error) {
	order, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return 0, err
	}
	return pagoMovilAmountBs(order, bcvRateFromPaymentDetails(order)), nil
}

func (s *orderPaymentSubject) CustomerID(ctx context.Context, referenceID uuid.UUID) (uuid.UUID, error) {
	order, err := s.orders.SearchOrdersByID(ctx, referenceID)
	if err != nil {
		return uuid.Nil, err
	}
	return order.CustomerID, nil
}

func (s *orderPaymentSubject) OnApproved(ctx context.Context, referenceID uuid.UUID) error {
	order, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	if err := s.syncPaymentDetails(ctx, order, referenceID); err != nil {
		return err
	}
	_, err = s.releaseToShop(ctx, order)
	return err
}

func (s *orderPaymentSubject) OnOverpaid(ctx context.Context, referenceID uuid.UUID, diffBs float64) error {
	// El crédito de balance ya lo hizo payments/app antes de llamar acá.
	// ApproveOverpay también libera al shop tras acreditar el excedente,
	// igual que Approve.
	return s.OnApproved(ctx, referenceID)
}

func (s *orderPaymentSubject) OnRejected(ctx context.Context, referenceID uuid.UUID, reason string) error {
	order, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	if err := s.syncPaymentDetails(ctx, order, referenceID); err != nil {
		return err
	}
	balanceUsedBs := balanceAmountBs(order)
	if err := s.refundBalanceUsed(ctx, order, balanceUsedBs); err != nil {
		return err
	}
	if err := s.orders.UpdateOrderStatus(ctx, order.ID, string(models.StatusRejected)); err != nil {
		return err
	}
	order.Status = string(models.StatusRejected)
	if s.mgr != nil {
		_ = s.mgr.BroadcastOrderToCustomer(*order)
	}
	return nil
}
```

  - `loadInPaymentReview` reemplaza a `loadPending`: solo valida `order.Status == StatusPaymentReview` (la validación de `review.Status` ya la hizo `payments/app` antes de invocar el subject).
  - Se conservan tal cual (movidos a este archivo): `releaseToShop`, `syncPaymentDetails` (ahora toma el review vía `s.reviews.GetByID`), `balanceAmountBs`, `pagoMovilAmountBs`, `paymentSplitFromDetails`, `cashRefundablePMBs`, `refundBalanceUsed`, `bcvRateFromPaymentDetails`.

- **`internal/orders/app/payment_proof_alert_worker.go`**: pasa a depender de `paymentsPorts.PaymentReviewsRepo` directo:
```go
type PaymentProofAlertWorker struct {
	reviewsRepo paymentsPorts.PaymentReviewsRepo
	orders      ports.OrdersRepo
	stream      ports.RedisStream
}

func (w *PaymentProofAlertWorker) tick(ctx context.Context) {
	alertBefore := time.Now().Add(-paymentsModels.PaymentProofAlertInterval)
	list, _ := w.reviewsRepo.ListPendingForAlert(ctx, paymentsModels.ReferenceTypeOrder, alertBefore)
	for _, review := range list {
		ok, _ := w.reviewsRepo.MarkAlerted(ctx, review.ID, time.Now(), alertBefore)
		if !ok {
			continue
		}
		order, err := w.orders.SearchOrdersByID(ctx, review.ReferenceID)
		if err != nil {
			continue
		}
		var event orderModels.DataEventPaymentProofPending // SE MANTIENE en orders/domain/models, sin generalizar
		event.BuildFrom(*order, review, minutesPending(review))
		_ = w.stream.PublisherEvent(ctx, &event)
	}
}
```
  Preserva **exactamente** el payload actual (`shop_id`, `customer_id`, `total_usd`) para el consumidor externo (notificaciones) — solo cambia de dónde lee el review (`review.ReferenceID` en vez de `review.OrderID`).

- **`internal/orders/app/order.go`**:
  - Constructor: `reviewsRepo ports.PaymentReviewsRepo` → tipo de `payments` (ver alias abajo); `staffHub`, `customerBalance` ídem.
  - Construcción de `PaymentReview`/`PaymentReviewBroadcast` (hoy ~líneas 589-627): usar `paymentsModels.PaymentReview{ReferenceID: orderDB.ID, ReferenceType: paymentsModels.ReferenceTypeOrder, ...}`. **Sigue llamando `o.reviewsRepo.Create` directo** (no pasa por `payments/app.CreateReview` — ver justificación en sección 4.8).
  - Todas las llamadas a `o.customerBalance.*` pasan `orderModel.ID, paymentsModels.ReferenceTypeOrder` como los dos primeros parámetros de referencia.

- **`internal/orders/domain/ports/payments.go`** (nuevo, chico) — type aliases para minimizar el diff en el resto de `orders`:
```go
package ports

import paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"

type PaymentReviewsRepo = paymentsPorts.PaymentReviewsRepo
type StaffReviewHub = paymentsPorts.StaffReviewHub
type CustomerBalanceCreditor = paymentsPorts.CustomerBalanceCreditor
```

- **`internal/orders/domain/models/detailsOrder.go`**: `PaymentDetails.Status` pasa a `paymentsModels.PaymentReviewStatus` (o alias `type PaymentReviewStatus = paymentsModels.PaymentReviewStatus` en `orders/domain/models` para no tocar cada uso de `models.ReviewPendingReview` etc. dentro de `orders`).

- **`api/router/groups/orders.go`**: quitar el handler de staff reviews del struct/constructor y la ruta `GET /staff/reviews`.

### `internal/users/`

- **`domain/models/finance.go`**: `Transaction.OrderID *uuid.UUID` → `ReferenceID *uuid.UUID; ReferenceType *string` (no mapea `order_id`, sección 3.4).
- **`domain/ports/finance.go`**: `CustomerFinanceRepo`/`CustomerFinanceApp` — mismos 6 métodos que `paymentsPorts.CustomerBalanceCreditor`, +`referenceType string` en cada uno (el cast estructural en DI depende de que ambos lados tengan EXACTAMENTE el mismo set de firmas).
- **`domain/models/refund.go`** (`BalanceRefundRequest`): `OrderID *uuid.UUID` → `ReferenceID *uuid.UUID; ReferenceType *string`.
- **`app/finance.go`** (`customerFinanceApp`): todos los métodos de crédito/débito agregan `referenceType string`, lo persisten en `Transaction.ReferenceType`. `CreateRefundRequest`: `req.OrderID = tx.OrderID` → `req.ReferenceID = tx.ReferenceID; req.ReferenceType = tx.ReferenceType`.
- **`repo/finance.go`**: filtros/inserts a `reference_id`/`reference_type` en vez de `order_id`.
- **DTOs de customer** (`dto.CustomerTransactionItem`, refund requests): **shape JSON no cambia** — se sigue poblando `order_id` (omitempty) cuando `ReferenceType=="order"`. Son endpoints customer-facing (app móvil) que el usuario no pidió romper — solo el WS de Staff cambia directo.

### `internal/finance/` (staff payouts riders/shops)

- **`domain/models/balance.go`** (`Transaction`, lado customer/staff-payout — no confundir con `riders/domain/models.Transaction`, que NO se toca): mismo cambio `OrderID→ReferenceID+ReferenceType`. Confirmado por grep que nunca se popula en `internal/finance/app|repo/staff.go` hoy — diff mecánico, sin riesgo de comportamiento.

### `internal/riders/` — sin cambios (confirmado fuera de alcance, sección 2.5).

### `cmd/server/providers/di.go` + `cmd/server/main.go`

```go
// di.go — nuevo bloque
_ = Container.Provide(groups.NewPaymentsGroup)
_ = Container.Provide(paymentsApp.NewSubjectRegistry)
_ = Container.Provide(func(r *paymentsApp.SubjectRegistry) paymentsPorts.PaymentSubjectRegistry { return r })
_ = Container.Provide(paymentsApp.NewStaffReviewHub)
_ = Container.Provide(func(h *paymentsApp.StaffReviewHubImpl) paymentsPorts.StaffReviewHub { return h })
_ = Container.Provide(paymentsRepo.NewPaymentReviewsRepo)
_ = Container.Provide(func(fa usersPorts.CustomerFinanceApp) paymentsPorts.CustomerBalanceCreditor { return fa })
_ = Container.Provide(paymentsApp.NewPaymentReviewsApp)
_ = Container.Provide(paymentsHandler.NewStaffReviewsHandler)

_ = Container.Provide(appOrders.NewOrderPaymentSubject)
_ = Container.Provide(appOrders.NewPaymentProofAlertWorker)
```

```go
// main.go — dentro del container.Invoke existente, junto a los demás setters post-construcción
registry.Register(paymentsModels.ReferenceTypeOrder, orderSubject)
hub.SetSnapshotProvider(orderApp.BuildStaffSnapshot) // nuevo método en orders/app, hace el join con Order
```

### Router

- **`api/router/groups/payments.go`** (nuevo):
```go
func (p *paymentsGroup) Resource(g *echo.Echo) {
	group := g.Group("/payments")
	group.GET("/staff/reviews", p.handlerStaffReviews.HandleWebSocket, p.middle.Auth, p.middle.Require(models.Staff))
}
```
- **`api/router/router.go`**: agregar `paymentsGroup groups.PaymentsGroup` al struct/constructor/`Init()`.

## 7. DTOs / contrato del WS

**Entrantes** (sin cambio de shape, solo de endpoint — `wss://.../payments/staff/reviews`):
```json
{"type":"approve","review_id":"...","reference_last4":"1234"}
{"type":"approve_overpay","review_id":"...","observed_amount_bs":123.45,"reference_last4":"1234"}
{"type":"reject_underpaid","review_id":"...","observed_amount_bs":100,"reference_last4":"1234"}
{"type":"reject_invalid","review_id":"...","reason":"..."}
```

**Salientes**:
- `{"type":"connected"}` — sin cambios.
- Snapshot al conectar + `payment_review_update` (broadcast en creación y en resolución): mismo shape que hoy + `reference_type`.
- `action_ok`: `{"type":"action_ok","action":"approve","review_id":"...","reference_id":"...","reference_type":"order"}` (sin `order_id`/`order_status`).

**`POST/GET /users/balance/refund-requests`**: sin cambios de shape (sección 6, `internal/users`).

## 8. Plan de tests

- **`internal/test/payments/app/`** (nuevo): CAS del review, dispatch por `ReferenceType` vía mock de `PaymentSubjectRegistry`, duplicado de `reference_last4` auto-rechaza, `ReferenceType` desconocido propaga error, `ApproveOverpay`/`RejectUnderpaid` acreditan con el `referenceType` correcto usando `subject.CustomerID`.
- **`internal/test/orders/app/`**: reescribir tests de `payment_reviews_test.go` → `order_payment_subject_test.go`, probando `OnApproved`/`OnOverpaid`/`OnRejected` con el MISMO comportamiento observable de hoy (releaseToShop, refundBalanceUsed, transiciones de estado, `order.Status != StatusPaymentReview` → error). `payment_proof_alert_test.go` actualizado contra `paymentsPorts.PaymentReviewsRepo` mockeado.
- **`internal/test/users/app/finance_test.go`**: firmas con `referenceType` explícito, verificar `Transaction.ReferenceID/ReferenceType` en vez de `OrderID`; `GetBalance` sigue poblando `order_id` en el DTO cuando corresponde.
- **`internal/riders/...`**: sin cambios de test — correr `go test ./internal/riders/...` para confirmar que sigue verde (columna agregada a una tabla compartida, ninguna removida).
- **`internal/test/payments/handler/`** (nuevo): WS de staff reviews, patrón `buildhttptest.SetupControllerCase`.
- Mocks: agregar entrada `internal/payments/domain/ports` a `.mockery.yaml` (`dir: internal/mocks/payments/domain/ports`), regenerar todo con `mockery` (usar `GOTOOLCHAIN=go1.26.4` — ver nota de entorno abajo).

## 9. Orden de ejecución recomendado

1. `payments/domain/models` + `dtos` + `ports` — sin dependencias, compila solo.
2. Migraciones SQL (aplicar con `goose ... up`, validar).
3. `payments/repo`.
4. `payments/app` (CAS genérico, registry, hub).
5. `internal/users` (Transaction/BalanceRefundRequest/CustomerFinanceApp/CustomerFinanceRepo) — **antes** que orders, porque orders depende de `usersPorts.CustomerFinanceApp` vía DI.
6. `internal/finance` (Transaction de riders/shops-payout) — puede ir en paralelo con el paso 5.
7. `internal/orders` (`orderPaymentSubject`, `order.go`, alert worker, eliminar archivos obsoletos).
8. `payments/handler` + `api/router/groups/payments.go` + ajustar `orders.go`/`router.go`.
9. DI (`di.go` + `main.go`).
10. Tests + mocks (regenerar todo con `mockery`).
11. Swagger (`swag init`) + Postman (mover request de `orders` a colección nueva `payments`) + actualizar `docs/frontend-payment-review.md` con el nuevo path/shape del WS.

Compilar/testear (`go build ./...`, `go vet ./...`, `go test ./...`) después de cada bloque, no solo al final — el cambio es grande.

## 10. Verificación end-to-end

1. `go build ./... && go vet ./... && go test ./...` en verde.
2. `golangci-lint`/`mockery`/`swag` con `GOTOOLCHAIN=go1.26.4` — el `go` local de Arch/CachyOS (1.27.0) es más nuevo que lo que pide `go.mod` (1.26.4) y rompe esas herramientas (parser de `math/rand/v2`); ya se resolvió una vez en la sesión de #182, mismo fix aplica acá.
3. Auditoría manual (arquitectura + seguridad, mismo criterio que #182) antes de comitear — este cambio toca dinero real (balance, créditos, reembolsos), amerita el mismo nivel de revisión. Prestar atención especial a: ¿el `reference_last4` sigue siendo único GLOBAL (no por tipo)? ¿`CustomerID` del subject no puede ser suplantado por un `referenceID` de otro dominio? ¿los montos de crédito/débito usan siempre el `referenceType` correcto (no se puede acreditar balance de una orden marcándola como si fuera un trip)?
4. Prueba manual/e2e contra Staging: Staff conectado a `GET /payments/staff/reviews` ve una orden pendiente, la aprueba (releaseToShop ocurre, balance no se toca), la aprueba con excedente (balance se acredita, releaseToShop ocurre), la rechaza por pago insuficiente (balance se acredita parcial, orden pasa a Rejected, balance usado al pagar se refunda), comprobante duplicado se auto-rechaza. Regresión completa del flujo de orders existente — el comportamiento observable para customer/shop no debe cambiar.
5. Confirmar `go test ./internal/riders/...` sigue verde sin modificaciones.
6. Avisar al equipo/repo del CRM (`Encargalo.app-CRM`) sobre el cambio de path y shape del WS — no es parte de este PR pero bloquea que Staff pueda operar tras el deploy si no se coordina.

### Archivos críticos
- `internal/payments/app/payment_reviews.go` (nuevo)
- `internal/payments/domain/ports/subject.go` (nuevo)
- `internal/orders/app/order.go`
- `internal/orders/app/payment_subject.go` (nuevo, reemplaza a `payment_reviews.go`)
- `internal/users/app/finance.go`
- `cmd/server/providers/di.go` + `cmd/server/main.go`
- `internal/shared/adapters/postgres/migrations/00074_payments_schema.sql` (nuevo)
