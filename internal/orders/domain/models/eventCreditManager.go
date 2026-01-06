package models

import "github.com/google/uuid"

type CreditManager struct {
	Amount  float64   `json:"amount"`
	OrderID uuid.UUID `json:"order_id"`
}
