package dtos

import "github.com/google/uuid"

type ShopsResponse []ShopResponse

type ShopResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Address       string    `json:"address"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	HomePhone     string    `json:"home_phone"`
	Logo          string    `json:"logo"`
	Banner        string    `json:"banner"`
	Opened        bool      `json:"opened"`
	Type          string    `json:"type"`
	Specialty     string    `json:"specialty"`
	Score         float64   `json:"score"`
	DeliveryFee   float64   `json:"delivery_fee"`
	DeliveryFeeBs float64   `json:"delivery_fee_bs"`
	Time          int       `json:"time"`
	// HasLaunchPromo indica si el negocio está dentro de su ventana de
	// lanzamiento (día 1): los productos se muestran con 50% de descuento.
	HasLaunchPromo bool `json:"has_launch_promo"`
}

func (s *ShopsResponse) Add(shop ShopResponse) {
	*s = append(*s, shop)
}
