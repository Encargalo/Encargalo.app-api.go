package dtos

import (
	"context"

	"github.com/google/uuid"
)

type RegisterFlavor struct {
	ShopID      uuid.UUID `json:"shop_id" swaggerignore:"true"`
	CategoryID  uuid.UUID `json:"category_id" validate:"required" example:"b01e3f7a-ff6e-45c0-842b-7f89ab48f9e2"`
	Name        string    `json:"name" validate:"required" example:"Picante"`
	Description string    `json:"description" example:"Con salsa picante casera"`
}

func (r *RegisterFlavor) Validate() error {
	_ = conform.Struct(context.Background(), r)
	return validate.Struct(r)
}
