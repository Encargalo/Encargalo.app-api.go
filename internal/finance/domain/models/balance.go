package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	CurrencyVES = "VES"
	CurrencyUSD = "USD"
)

type MovementType string

const (
	MovementTypeStaffPayout      MovementType = "staff_payout"
	MovementTypeAdjustmentCredit MovementType = "adjustment_credit"
	MovementTypeAdjustmentDebit  MovementType = "adjustment_debit"
)

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
	BcvRate       *float64   `bun:"bcv_rate,nullzero"`
	MovementType  string     `bun:"movement_type"`
	PaymentMethod *string    `bun:"payment_method"`
	DistanceKm    *float64   `bun:"distance_km"`
	StaffID       *uuid.UUID `bun:"staff_id,nullzero"`
	Notes         *string    `bun:"notes,nullzero"`
	ProofURL      *string    `bun:"proof_url,nullzero"`
	CreatedAt     time.Time  `bun:"created_at,nullzero"`
}

type PayableRow struct {
	EntityID  uuid.UUID `bun:"entity_id"`
	Amount    float64   `bun:"amount"`
	Currency  string    `bun:"currency"`
	UpdatedAt time.Time `bun:"updated_at"`
	Name      string    `bun:"name"`
	SurName   *string   `bun:"sur_name"`
	Phone     *string   `bun:"phone"`
}
