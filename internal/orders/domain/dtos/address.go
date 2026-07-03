package dtos

import "context"

type Address struct {
	Address      string  `json:"address" validate:"required" example:"Calle 123 # 45-67"`
	Latitude     float64 `json:"latitude" validate:"required,latitude" example:"37.7749"`
	Longitude    float64 `json:"longitude" validate:"required,longitude" example:"-74.081750"`
	Instructions string  `json:"instructions" validate:"omitempty" example:"Piso 3, apto 301"`
}

func (c *Address) Validate() error {
	_ = conform.Struct(context.Background(), c)
	return validate.Struct(c)
}
