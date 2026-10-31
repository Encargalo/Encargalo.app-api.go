package dtos

import "context"

// ListShopsQuery es el query de GET /shops/all: coordenadas + type opcional.
// Se separa de Coords para no contaminar GetShopsBy.
type ListShopsQuery struct {
	Latitude  float64 `query:"lat" validate:"required,latitude"`
	Longitude float64 `query:"lon" validate:"required,longitude"`
	Type      string  `query:"type" validate:"omitempty,oneof=Restaurant Supermarket Turbo"`
	Specialty string  `query:"specialty" validate:"omitempty,oneof=General Panadería Carnicería Charcutería Frutería Licorería Pizzería 'Comida Rápida' 'Comida China' Heladería Cafetería"`
}

func (q *ListShopsQuery) Validate() error {
	_ = conform.Struct(context.Background(), q)
	return validate.Struct(q)
}

func (q ListShopsQuery) Coords() Coords {
	return Coords{Latitude: q.Latitude, Longitude: q.Longitude}
}
