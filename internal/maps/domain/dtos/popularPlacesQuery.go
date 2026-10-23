package dtos

import (
	"context"

	"github.com/go-playground/validator/v10"
)

var mapsValidate = validator.New()

type PopularPlacesQuery struct {
	// Sin `required`: en un float64 ese tag rechaza el cero, y lat=0 (ecuador) y lng=0 (Greenwich) son coordenadas válidas — la presencia se verifica en el handler.
	Latitude  float64 `query:"lat" validate:"latitude"`
	Longitude float64 `query:"lng" validate:"longitude"`
	// max=80 es el techo real del upstream: 20 lugares por grupo de categorías y son 4 grupos.
	Limit int `query:"limit" validate:"omitempty,min=1,max=80"`
}

func (q *PopularPlacesQuery) Validate() error {
	return mapsValidate.StructCtx(context.Background(), q)
}
