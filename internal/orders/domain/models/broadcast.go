package models

import "github.com/google/uuid"

type OrderBroadcast struct {
	Type       string   `json:"type"`
	Order      Order    `json:"order"`
	SaleAmount *float64 `json:"sale_amount,omitempty"`
	Commission *float64 `json:"commission,omitempty"`
	NetAmount  *float64 `json:"net_amount,omitempty"`
}

// RiderLocationBroadcast es el mensaje WS con la ubicación en vivo del rider de una orden.
type RiderLocationBroadcast struct {
	Type      string    `json:"type"`
	OrderID   uuid.UUID `json:"order_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	UpdatedAt string    `json:"updated_at"`
}
