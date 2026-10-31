package dtos

import (
	"time"

	"github.com/google/uuid"
)

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
	// OpensAt/ClosesAt se calculan a partir del horario semanal configurado
	// (business.schedules): ClosesAt solo si Opened=true, OpensAt solo si
	// Opened=false. Ambos nil si el shop no tiene horario configurado.
	OpensAt  *time.Time `json:"opens_at,omitempty"`
	ClosesAt *time.Time `json:"closes_at,omitempty"`
}

func (s *ShopsResponse) Add(shop ShopResponse) {
	*s = append(*s, shop)
}
