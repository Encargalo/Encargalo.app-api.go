package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UpdateFlavor struct {
	bun.BaseModel `bun:"table:products.flavors"`

	ID          uuid.UUID `bun:"id,pk,type:uuid"`
	Name        string    `bun:"name,notnull"`
	Description *string   `bun:"description"`
}

func (u *UpdateFlavor) BuildToModel(flavor dtos.UpdateFlavor, flavorID uuid.UUID) {
	u.ID = flavorID
	u.Name = flavor.Name
	u.Description = flavor.Description
}
