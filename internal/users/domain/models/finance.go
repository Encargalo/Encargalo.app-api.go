package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MovementType string

const (
	MovementTypePaymentOverage          MovementType = "payment_overage"
	MovementTypePaymentPartialCredit    MovementType = "payment_partial_credit"
	MovementTypeOrderRejectedRefund     MovementType = "order_rejected_refund"
	MovementTypeBalancePayment          MovementType = "balance_payment"
	MovementTypeBalancePaymentRefund    MovementType = "balance_payment_refund"
	MovementTypeBalanceRefundHold       MovementType = "balance_refund_hold"       // pending: saldo retenido
	MovementTypeBalanceRefundProcessing MovementType = "balance_refund_processing" // accepted: en proceso de pago
	MovementTypeBalanceRefundPayout     MovementType = "balance_refund_payout"     // completed: ya pagado
	MovementTypeBalanceRefundRestore    MovementType = "balance_refund_restore"    // rejected: saldo restaurado
	EntityTypeCustomer                  string       = "customer"
)

// CurrencyVES es la moneda nativa del balance de customers: todo movimiento
// se registra ya convertido a bolívares con la tasa BCV vigente al momento
// del movimiento (ver Transaction.BcvRate). El catálogo de productos y el
// total de la orden se siguen calculando en USD; la conversión ocurre una
// sola vez, al debitar/acreditar el balance.
const CurrencyVES = "VES"

type Balance struct {
	bun.BaseModel `bun:"table:finance.balances"`

	EntityID   uuid.UUID `bun:"entity_id,pk,type:uuid"`
	EntityType string    `bun:"entity_type,pk"`
	Amount     float64   `bun:"amount"`
	Currency   string    `bun:"currency"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

type Transaction struct {
	bun.BaseModel `bun:"table:finance.transactions"`

	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	EntityID      uuid.UUID  `bun:"entity_id"`
	EntityType    string     `bun:"entity_type"`
	ReferenceID   *uuid.UUID `bun:"reference_id,nullzero"`
	ReferenceType *string    `bun:"reference_type,nullzero"`
	Amount        float64    `bun:"amount"`
	Currency      string     `bun:"currency"`
	// BcvRate es la tasa usada para convertir el monto original en USD (precio
	// del pedido) al Bs registrado en Amount. Nil en movimientos que ya nacen
	// en Bs y no requieren conversión (ej. RefundBalanceRestore).
	BcvRate         *float64   `bun:"bcv_rate,nullzero"`
	MovementType    string     `bun:"movement_type"`
	PaymentMethod   *string    `bun:"payment_method"`
	DistanceKm      *float64   `bun:"distance_km"`
	RefundablePMBs  *float64   `bun:"refundable_pm_bs,nullzero"`
	RefundRequestID *uuid.UUID `bun:"refund_request_id,nullzero"`
	CreatedAt       time.Time  `bun:"created_at,nullzero"`
}
