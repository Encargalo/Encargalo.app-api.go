package dtos

import (
	"context"
)

type UpdateItem struct {
	Name        string `json:"name" validate:"required" example:"Pizza Hawaiana Mediana"`
	Price       int    `json:"price" validate:"required" example:"32000"`
	Description string `json:"description" validate:"required" example:"Pizza mediana con jamón, piña y queso mozzarella sobre una base de salsa napolitana."`
	HasFlavors  *bool  `json:"has_flavors" validate:"required" example:"true"`
}

func (u *UpdateItem) Validate() error {
	_ = conform.Struct(context.Background(), u)
	return validate.Struct(u)
}
