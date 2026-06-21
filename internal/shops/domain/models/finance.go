package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	EntityTypeShop = "shop"
	ShopShare      = 0.90
)

type ShopMovementType string

const (
	MovementTypeSaleDeliveryBank ShopMovementType = "sale_delivery_bank"
	MovementTypeSaleDeliveryCash ShopMovementType = "sale_delivery_cash"
)

type ShopBalance struct {
	bun.BaseModel `bun:"table:finance.balances"`

	EntityID   uuid.UUID `bun:"entity_id,pk,type:uuid"`
	EntityType string    `bun:"entity_type,pk"`
	Amount     float64   `bun:"amount"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

type ShopTransaction struct {
	bun.BaseModel `bun:"table:finance.transactions"`

	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	EntityID      uuid.UUID  `bun:"entity_id"`
	EntityType    string     `bun:"entity_type"`
	OrderID       *uuid.UUID `bun:"order_id,nullzero"`
	Amount        float64    `bun:"amount"`
	MovementType  string     `bun:"movement_type"`
	PaymentMethod *string    `bun:"payment_method"`
	CreatedAt     time.Time  `bun:"created_at"`
}
