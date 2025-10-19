package dto

import (
	"context"

	"github.com/google/uuid"
)

type Addresses []Address

type Address struct {
	ID        uuid.UUID `json:"id"`
	Alias     string    `json:"alias" validate:"required" example:"Casa principal"`
	Address   string    `json:"address" validate:"required" example:"Calle 123 # 45-67"`
	Reference string    `json:"reference" validate:"required" example:"Frente al parque de los niños"`
	Cords     Coords    `json:"coords" validate:"required"`
}

type Coords struct {
	Latitude  float64 `json:"lat" validate:"required,latitude" example:"4.609710"`
	Longitude float64 `json:"long" validate:"required,longitude" example:"-74.081750"`
}

func (c *Address) Validate() error {
	_ = conform.Struct(context.Background(), c)
	return validate.Struct(c)
}

func (c *Addresses) Add(address Address) {
	*c = append(*c, address)
}
