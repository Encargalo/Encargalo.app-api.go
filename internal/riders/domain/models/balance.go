package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MovementType string
type Zone string

const (
	MovementTypeRideBank MovementType = "ride_bank"
	MovementTypeWithdraw MovementType = "withdrawal"

	ZoneNormal              Zone = "normal"
	ZoneWithdrawalAvailable Zone = "withdrawal_available"

	WithdrawalMinBalance float64 = 15.0
	WithdrawalCooldown           = 24 * time.Hour
	RateDay              float64 = 0.40
	RateNight            float64 = 0.60
	RiderShare           float64 = 0.80
	NightHour            int     = 22
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
	CreatedAt     time.Time  `bun:"created_at,nullzero"`
}
