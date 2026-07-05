package dtos

import (
	"context"

	"github.com/google/uuid"
)

type SearchShopsQuery struct {
	Q         string  `query:"q"   validate:"required"`
	Latitude  float64 `query:"lat" validate:"required,latitude"`
	Longitude float64 `query:"lon" validate:"required,longitude"`
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

type MatchedItemRule struct {
	ID           uuid.UUID `json:"id"`
	ItemID       uuid.UUID `json:"item_id"`
	RuleKey      string    `json:"rule_key"`
	RuleValue    int       `json:"rule_value"`
	SelectorType string    `json:"selector_type"`
}

type MatchedSchedule struct {
	DayOfWeek *int16 `json:"day_of_week,omitempty"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Enabled   bool   `json:"enabled"`
}

type MatchedProductItem struct {
	ID          uuid.UUID         `json:"id"`
	ShopID      uuid.UUID         `json:"shop_id"`
	Name        string            `json:"name"`
	Ref         float64           `json:"ref"`
	Image       string            `json:"image"`
	Description string            `json:"description"`
	Score       float32           `json:"score"`
	HasFlavors  bool              `json:"has_flavors"`
	Rules       []MatchedItemRule `json:"rules,omitempty"`
	Schedules   []MatchedSchedule `json:"schedules,omitempty"`
}

type SearchShopResponse struct {
	ShopResponse
	MatchedProducts []MatchedProductItem `json:"matched_products,omitempty"`
}

type SearchShopsResult []SearchShopResponse
