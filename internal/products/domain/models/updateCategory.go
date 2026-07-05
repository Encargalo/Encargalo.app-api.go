package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UpdateCategory struct {
	bun.BaseModel `bun:"table:products.categories"`

	ID              uuid.UUID  `bun:"id,pk,type:uuid"`
	Name            string     `bun:"name,notnull"`
	MacroCategoryID *uuid.UUID `bun:"macro_category_id"`
}

func (u *UpdateCategory) BuildToModel(category dtos.UpdateCategory, categoryID uuid.UUID) {
	u.ID = categoryID
	u.Name = category.Name
	u.MacroCategoryID = category.MacroCategoryID
}
