package dtos

import (
	"github.com/google/uuid"
)

type ItemsResponse []ItemResponse

type ItemResponse struct {
	ID         uuid.UUID `json:"id" example:"7e441237-a818-42d2-bb54-9b8747198305"`
	ShopID     uuid.UUID `json:"shop_id" example:"d33aaf08-2c43-41c4-b2e7-882b019edb1e"`
	CategoryID uuid.UUID `json:"category_id" example:"b01e3f7a-ff6e-45c0-842b-7f89ab48f9e2"`
	Name       string    `json:"name" example:"Pizza Hawaiana Mediana"`
	Ref        float64   `json:"ref" example:"14"`
	Bs         float64   `json:"bs" example:"7000"`
	// EffectiveRef/EffectiveBs son el precio final que paga el cliente tras
	// aplicar, en orden, el descuento propio del negocio y luego la promo de
	// lanzamiento (día 1). Ref/Bs siempre muestran el precio de catálogo
	// original, sin descuentos, para no romper compatibilidad.
	EffectiveRef    float64            `json:"effective_ref" example:"14"`
	EffectiveBs     float64            `json:"effective_bs" example:"7000"`
	HasShopDiscount bool               `json:"has_shop_discount" example:"false"`
	HasLaunchPromo  bool               `json:"has_launch_promo" example:"false"`
	Image           string             `json:"image" example:"https://cdn.encargalo.app/items/pizza-hawaiana-mediana.jpg"`
	Description     string             `json:"description" example:"Pizza mediana con jamón, piña y queso mozzarella sobre una base de salsa napolitana."`
	Score           float32            `json:"score" example:"4.7"`
	HasFlavors      bool               `json:"has_flavors" example:"true"`
	Rules           []ItemRule         `json:"rules,omitempty"`
	Schedules       []ScheduleResponse `json:"schedules,omitempty"`
}

func (i *ItemsResponse) Add(item ItemResponse) {
	*i = append(*i, item)
}
