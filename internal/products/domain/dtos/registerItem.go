package dtos

import (
	"context"

	"github.com/go-playground/mold/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validate = validator.New()
	conform  = modifiers.New()
)

type RegisterItem struct {
	ShopID      uuid.UUID `json:"shop_id" example:"d33aaf08-2c43-41c4-b2e7-882b019edb1e" swaggerignore:"true"`
	CategoryID  uuid.UUID `json:"category_id" validate:"required" example:"b01e3f7a-ff6e-45c0-842b-7f89ab48f9e2"`
	Name        string    `json:"name" validate:"required" example:"Pizza Hawaiana Mediana"`
	Price       int       `json:"price" validate:"required" example:"32000"`
	Image       string    `json:"image" validate:"required" example:"data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEBLAEsAAD/4QC8RXhpZgAASUkqAAgAAAAGABIBAwABAAAAAQAAABoBBQABAAAAVgAAABsBBQABAAAAXgAAACgBAwABAAAAAgAAABMCAwABAAAA"`
	Description string    `json:"description" validate:"required" example:"Pizza mediana con jamón, piña y queso mozzarella sobre una base de salsa napolitana."`
	HasFlavors  *bool     `json:"has_flavors" validate:"required" example:"true"`
}

func (r *RegisterItem) Validate() error {
	_ = conform.Struct(context.Background(), r)
	return validate.Struct(r)
}
