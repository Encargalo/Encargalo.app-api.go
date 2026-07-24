package models

import "github.com/google/uuid"

type Addition struct {
	ID     uuid.UUID `json:"id"`
	ShopID uuid.UUID `json:"shop_id"`
	Name   string    `json:"name"`
	Price  int       `json:"price"`
}
