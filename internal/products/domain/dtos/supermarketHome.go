package dtos

import (
	"context"

	"github.com/google/uuid"
)

type SupermarketHomeCoords struct {
	Latitude  float64 `query:"lat" validate:"required,latitude"`
	Longitude float64 `query:"lon" validate:"required,longitude"`
}

func (c *SupermarketHomeCoords) Validate() error {
	_ = conform.Struct(context.Background(), c)
	return validate.Struct(c)
}

// HomeEntry es la forma plana (un producto + su shop + su macro/micro) que
// devuelve el repo, cacheada junto a Schedules/DeliveryFee. El app layer la
// filtra en vivo por horario y la agrupa en SupermarketHomeResponse.
type HomeEntry struct {
	CategoryName string
	Shop         HomeShop
	Product      HomeProduct
	Schedules    []ScheduleResponse
}

type HomeEntriesResponse []HomeEntry

type SupermarketHomeResponse []HomeMacroCategory

type HomeMacroCategory struct {
	MacroCategory   MacroCategoryResponse `json:"macro_category"`
	MicroCategories []HomeMicroCategory   `json:"micro_categories"`
}

type HomeMicroCategory struct {
	Name     string        `json:"name"`
	Products []HomeProduct `json:"products"`
}

type HomeProduct struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Ref   float64   `json:"ref"`
	Bs    float64   `json:"bs"`
	Image string    `json:"image"`
	Score float32   `json:"score"`
	Sold  int       `json:"sold"`
	Shop  HomeShop  `json:"shop"`
}

type HomeShop struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Logo          string    `json:"logo"`
	DeliveryFee   float64   `json:"delivery_fee"`
	DeliveryFeeBs float64   `json:"delivery_fee_bs"`
	Time          int       `json:"time"`
}
