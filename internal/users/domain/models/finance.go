package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MovementType string

const (
	MovementTypePaymentOverage       MovementType = "payment_overage"
	MovementTypePaymentPartialCredit MovementType = "payment_partial_credit"
	MovementTypeOrderRejectedRefund  MovementType = "order_rejected_refund"
	MovementTypeBalancePayment       MovementType = "balance_payment"
	MovementTypeBalancePaymentRefund MovementType = "balance_payment_refund"
	EntityTypeCustomer               string       = "customer"
)

type Balance struct {
	bun.BaseModel `bun:"table:finance.balances"`

	EntityID   uuid.UUID `bun:"entity_id,pk,type:uuid"`
	EntityType string    `bun:"entity_type,pk"`
	Amount     float64   `bun:"amount"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

type Transaction struct {
	bun.BaseModel `bun:"table:finance.transactions"`

	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	EntityID      uuid.UUID  `bun:"entity_id"`
	EntityType    string     `bun:"entity_type"`
	OrderID       *uuid.UUID `bun:"order_id,nullzero"`
	Amount        float64    `bun:"amount"`
	MovementType  string     `bun:"movement_type"`
	PaymentMethod *string    `bun:"payment_method"`
	DistanceKm    *float64   `bun:"distance_km"`
	CreatedAt     time.Time  `bun:"created_at"`
}
