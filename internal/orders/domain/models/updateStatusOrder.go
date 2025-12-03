package models

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UpdateStatusOrder struct {
	bun.BaseModel `bun:"table:orders.orders"`

	OrderID   uuid.UUID `json:"order_id"`
	NewStatus string    `json:"new_status"`
}
