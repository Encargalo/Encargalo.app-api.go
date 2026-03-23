package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UpdateItem struct {
	bun.BaseModel `bun:"table:products.items"`

	ID            uuid.UUID `bun:"id,pk,type:uuid"`
	Name          string    `bun:"name,notnull"`
	Price         int       `bun:"price,notnull"`
	Description   string    `bun:"description"`
	HasFlavors    bool      `bun:"has_flavors,notnull"`
	Image         string    `bun:"image"`
	ImagePublicID string    `bun:"image_public_id"`
	ImageHash     string    `bun:"image_hash"`
}

func (u *UpdateItem) BuildToModel(item dtos.UpdateItem, itemID uuid.UUID, hasFlavors bool, imageURL, imagePublicID, imageHash string) {
	u.ID = itemID
	u.Name = item.Name
	u.Price = item.Price
	u.Description = item.Description
	u.HasFlavors = hasFlavors
	u.Image = imageURL
	u.ImagePublicID = imagePublicID
	u.ImageHash = imageHash
}
