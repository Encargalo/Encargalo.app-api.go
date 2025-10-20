package models

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Items struct {
	bun.BaseModel `bun:"table:products.items" swaggerignore:"true"`

	ID     uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ShopID uuid.UUID `bun:"shop_id"`
	Name   string    `bun:"name" json:"name"`
	Price  int       `bun:"price" json:"price"`
}
