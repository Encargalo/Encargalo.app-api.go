package models

import (
	"github.com/google/uuid"
)

type DataProduceOrder struct {
	MessageType string    `bun:"type" json:"type"`
	ProductID   uuid.UUID `bun:"product_id" json:"product_id"`
	Quantity    int       `bun:"quantity" json:"quantity"`
}
