package models

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShopOrder struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name      string    `bun:"name" json:"name"`
	Phone     string    `bun:"home_phone" json:"phone"`
	Logo      string    `bun:"logo_image" json:"logo"`
	Address   string    `bun:"address" json:"address"`
	Latitude  float64   `bun:"latitude" json:"latitude"`
	Longitude float64   `bun:"longitude" json:"longitude"`
}
