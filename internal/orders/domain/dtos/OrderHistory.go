package dtos

import (
	"time"

	"github.com/google/uuid"
)

type OrderHistoryItem struct {
	ID            uuid.UUID `json:"id"`
	Number        int       `json:"number"`
	Status        string    `json:"status"`
	TotalPrice    float64   `json:"total_price"`
	DeliveryFee   float64   `json:"delivery_fee"`
	MethodPayment string    `json:"method_payment"`
	Address       string    `json:"address"`
	ShopID        uuid.UUID `json:"shop_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type OrderHistoryResponse []OrderHistoryItem
