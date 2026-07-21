package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShopOrder struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name      string    `bun:"name" json:"name"`
	Phone     string    `bun:"home_phone" json:"phone"`
	Logo      string    `bun:"logo_image" json:"logo"`
	Address   string    `bun:"address" json:"address"`
	Latitude  float64   `bun:"latitude" json:"latitude"`
	Longitude float64   `bun:"longitude" json:"longitude"`
	Opened    bool      `bun:"opened" json:"opened"`

	// LaunchPromoEndsAt/ZeroCommissionEndsAt anclan las promos de lanzamiento
	// del negocio a la primera vez que se marcó como abierto.
	LaunchPromoEndsAt    *time.Time `bun:"launch_promo_ends_at" json:"-"`
	ZeroCommissionEndsAt *time.Time `bun:"zero_commission_ends_at" json:"-"`
}

// HasActiveLaunchPromo indica si, en este instante, el pedido debería recibir
// el 50% de descuento al cliente (día 1 de vida del negocio).
func (s *ShopOrder) HasActiveLaunchPromo() bool {
	return s.LaunchPromoEndsAt != nil && time.Now().Before(*s.LaunchPromoEndsAt)
}

// HasActiveZeroCommission indica si, en este instante, el negocio debe
// recibir el 100% del subtotal al liquidar (días 1-8 de vida del negocio).
func (s *ShopOrder) HasActiveZeroCommission() bool {
	return s.ZeroCommissionEndsAt != nil && time.Now().Before(*s.ZeroCommissionEndsAt)
}
