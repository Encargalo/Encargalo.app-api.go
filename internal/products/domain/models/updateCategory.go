package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UpdateCategory struct {
	bun.BaseModel `bun:"table:products.categories"`

	ID                 uuid.UUID          `bun:"id,pk,type:uuid"`
	Name               string             `bun:"name,notnull"`
	MacroCategoryID    *uuid.UUID         `bun:"macro_category_id"`
	AvailabilityStatus AvailabilityStatus `bun:"availability_status,notnull"`
	Details            CategoryDetails    `bun:"details,type:jsonb,notnull"`
}

func (u *UpdateCategory) BuildToModel(category dtos.UpdateCategory, categoryID uuid.UUID, details CategoryDetails, current AvailabilityStatus) {
	u.ID = categoryID
	u.Name = category.Name
	macroCategoryID := category.MacroCategoryID
	u.MacroCategoryID = &macroCategoryID
	if category.AvailabilityStatus != nil {
		u.AvailabilityStatus = NormalizeAvailabilityStatus(*category.AvailabilityStatus)
	} else {
		u.AvailabilityStatus = current
	}
	u.Details = details
}
