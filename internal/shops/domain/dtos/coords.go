package dtos

import "context"

type Coords struct {
	Latitude  float64 `query:"lat" validate:"required,latitude"`
	Longitude float64 `query:"lon" validate:"required,longitude"`
}

func (c *Coords) Validate() error {
	_ = conform.Struct(context.Background(), c)
	return validate.Struct(c)
}
