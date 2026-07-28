package dtos

import (
	"context"

	"github.com/google/uuid"
)

type SearchShopsQuery struct {
	Q         string  `query:"q"   validate:"required"`
	Latitude  float64 `query:"lat" validate:"required,latitude"`
	Longitude float64 `query:"lon" validate:"required,longitude"`
	Type      string  `query:"type" validate:"omitempty,oneof=Restaurant Supermarket Turbo"`
}

func (q *SearchShopsQuery) Validate() error {
	_ = conform.Struct(context.Background(), q)
	return validate.Struct(q)
}

// SearchShopCandidate lleva el ShopResponse ya mapeado más el flag de
// coincidencia, para que la capa app decida qué productos cargar.
type SearchShopCandidate struct {
	ShopResponse
	MatchedByName bool `json:"matched_by_name"`
}

type MatchedSchedule struct {
	DayOfWeek *int16 `json:"day_of_week,omitempty"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Enabled   bool   `json:"enabled"`
}

type MatchedFlavorSelection struct {
	SelectorType string `json:"selector_type"`
	MinSelection int    `json:"min_selection"`
	MaxSelection int    `json:"max_selection"`
}

type MatchedAdditionSelection struct {
	SelectorType string `json:"selector_type"`
	MinSelection int    `json:"min_selection"`
	MaxSelection int    `json:"max_selection"`
}

type MatchedFlavor struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Available bool      `json:"available,omitempty"`
}

type MatchedAddition struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Ref       float64   `json:"ref"`
	Available bool      `json:"available,omitempty"`
}

// MatchedItemDetails espeja el details del catálogo (sabores + adiciones del item).
type MatchedItemDetails struct {
	Flavors               []MatchedFlavor   `json:"flavors,omitempty"`
	MinSelection          int               `json:"min_selection,omitempty"`
	MaxSelection          int               `json:"max_selection,omitempty"`
	Additions             []MatchedAddition `json:"additions,omitempty"`
	MaxAdditionsSelection int               `json:"max_additions_selection,omitempty"`
}

type MatchedProductItem struct {
	ID                uuid.UUID                 `json:"id"`
	ShopID            uuid.UUID                 `json:"shop_id"`
	Name              string                    `json:"name"`
	Ref               float64                   `json:"ref"`
	Image             string                    `json:"image"`
	Description       string                    `json:"description"`
	Score             float32                   `json:"score"`
	HasFlavors        bool                      `json:"has_flavors"`
	HasAdditions      bool                      `json:"has_additions"`
	FlavorSelection   *MatchedFlavorSelection   `json:"flavor_selection,omitempty"`
	AdditionSelection *MatchedAdditionSelection `json:"addition_selection,omitempty"`
	Details           *MatchedItemDetails       `json:"details,omitempty"`
	Schedules         []MatchedSchedule         `json:"schedules,omitempty"`
}

type SearchShopResponse struct {
	ShopResponse
	MatchedProducts []MatchedProductItem `json:"matched_products,omitempty"`
}

type SearchShopsResult []SearchShopResponse
