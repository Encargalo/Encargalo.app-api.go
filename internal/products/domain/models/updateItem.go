package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UpdateItem struct {
	bun.BaseModel `bun:"table:products.items"`

	ID          uuid.UUID `bun:"id,pk,type:uuid"`
	Name        string    `bun:"name,notnull"`
	Price       int       `bun:"price,notnull"`
	Description string    `bun:"description"`
	HasFlavors  *bool     `bun:"has_flavors,notnull"`
}

func (u *UpdateItem) BuildToModel(item dtos.UpdateItem, itemID uuid.UUID) {
	u.ID = itemID
	u.Name = item.Name
	u.Price = item.Price
	u.Description = item.Description
	u.HasFlavors = item.HasFlavors
}
