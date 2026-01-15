package dtos

import (
	"context"

	"github.com/google/uuid"
)

type OpenCloseShop struct {
	ShopID uuid.UUID `json:"shop_id"`
	Open   bool      `json:"open" validate:"required"`
}

func (o *OpenCloseShop) Validate() error {
	_ = conform.Struct(context.Background(), o)
	return validate.Struct(o)
}
