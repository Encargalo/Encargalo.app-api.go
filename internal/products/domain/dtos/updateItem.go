package dtos

import (
	"context"

	"github.com/google/uuid"
)

type UpdateItem struct {
	Name               string       `json:"name" validate:"required" example:"Pizza Hawaiana Mediana"`
	Price              int          `json:"price" validate:"required" example:"32000"`
	Description        string       `json:"description" validate:"required" example:"Pizza mediana con jamón, piña y queso mozzarella sobre una base de salsa napolitana."`
	Image              string       `json:"image" example:"data:image/jpeg;base64,/9j/4AAQSkZJRgABAQ..."`
	FlavorIDs          *[]uuid.UUID `json:"flavor_ids" example:"[\"a1b2c3d4-e5f6-7890-abcd-ef1234567890\"]"`
	MinSelection       *int         `json:"min_selection" example:"1"`
	MaxSelection       *int         `json:"max_selection" example:"2"`
	AdditionIDs        *[]uuid.UUID `json:"addition_ids" example:"[\"a1b2c3d4-e5f6-7890-abcd-ef1234567890\"]"`
	AvailabilityStatus *string      `json:"availability_status" validate:"omitempty,oneof=active hidden out_of_stock" example:"active"`
}

func (u *UpdateItem) Validate() error {
	_ = conform.Struct(context.Background(), u)
	return validate.Struct(u)
}
