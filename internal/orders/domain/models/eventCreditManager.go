package models

import "github.com/google/uuid"

type CreditManager struct {
	ShopID  uuid.UUID `json:"shop_id"`
	OrderID uuid.UUID `json:"order_id"`
	Amount  float64   `json:"amount"`
}
