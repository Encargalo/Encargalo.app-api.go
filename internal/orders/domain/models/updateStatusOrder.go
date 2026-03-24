package models

import "github.com/google/uuid"

type UpdateStatusOrder struct {
	OrderID   uuid.UUID `json:"order_id"`
	NewStatus string    `json:"new_status"`
}
