package dtos

import (
	"context"

	"github.com/google/uuid"
)

type TrendingCoords struct {
	Latitude  float64 `query:"lat" validate:"required,latitude"`
	Longitude float64 `query:"lon" validate:"required,longitude"`
}

func (c *TrendingCoords) Validate() error {
	_ = conform.Struct(context.Background(), c)
	return validate.Struct(c)
}

type TrendingResponse []TrendingEntry

type TrendingEntry struct {
	Shop    TrendingShop    `json:"shop"`
	Product TrendingProduct `json:"product"`
}

type TrendingShop struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Tag           string    `json:"tag"`
	Logo          string    `json:"logo"`
	Address       string    `json:"address"`
	Score         float64   `json:"score"`
	Type          string    `json:"type"`
	DeliveryFee   float64   `json:"delivery_fee"`
	DeliveryFeeBs float64   `json:"delivery_fee_bs"`
	Time          int       `json:"time"`
}

type TrendingProduct struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Ref         float64   `json:"ref"`
	Bs          float64   `json:"bs"`
	Image       string    `json:"image"`
	Description string    `json:"description"`
	Score       float32   `json:"score"`
	Sold        int       `json:"sold"`
}
