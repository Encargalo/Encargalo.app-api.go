package models

import (
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	DiscountTypePercentage = "percentage"
	DiscountTypeFixed      = "fixed"
)

type ItemDiscount struct {
	bun.BaseModel `bun:"table:products.item_discounts"`

	ID           uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ItemID       uuid.UUID  `bun:"item_id,notnull,type:uuid" json:"item_id"`
	ShopID       uuid.UUID  `bun:"shop_id,notnull,type:uuid" json:"shop_id"`
	DiscountType string     `bun:"discount_type,notnull" json:"discount_type"`
	Value        float64    `bun:"value,notnull" json:"value"`
	StartsAt     *time.Time `bun:"starts_at,nullzero" json:"starts_at,omitempty"`
	EndsAt       *time.Time `bun:"ends_at,nullzero" json:"ends_at,omitempty"`
	IsActive     bool       `bun:"is_active,default:true" json:"is_active"`
	CreatedAt    time.Time  `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time  `bun:"updated_at,default:current_timestamp" json:"updated_at"`
	DeletedAt    *time.Time `bun:"deleted_at,nullzero" json:"-"`
}

func (d *ItemDiscount) BuildToModel(discount dtos.RegisterItemDiscount) {
	d.ItemID = discount.ItemID
	d.ShopID = discount.ShopID
	d.DiscountType = discount.DiscountType
	d.Value = discount.Value
	d.StartsAt = discount.StartsAt
	d.EndsAt = discount.EndsAt
}

// IsCurrentlyActive determina si el descuento debe aplicarse ahora mismo,
// respetando el toggle manual (is_active) y la ventana de fechas opcional.
func (d *ItemDiscount) IsCurrentlyActive() bool {
	if d == nil || !d.IsActive || d.DeletedAt != nil {
		return false
	}
	now := time.Now()
	if d.StartsAt != nil && now.Before(*d.StartsAt) {
		return false
	}
	if d.EndsAt != nil && now.After(*d.EndsAt) {
		return false
	}
	return true
}

// Apply calcula el precio del negocio (en la misma unidad que Item.Price)
// tras restar este descuento, sin permitir precios negativos.
func (d *ItemDiscount) Apply(price float64) float64 {
	var discounted float64
	if d.DiscountType == DiscountTypePercentage {
		discounted = price * (1 - d.Value/100)
	} else {
		discounted = price - d.Value
	}
	if discounted < 0 {
		discounted = 0
	}
	return money.RoundUSD(discounted)
}

func (d *ItemDiscount) ToDomainDTO() dtos.ItemDiscountResponse {
	return dtos.ItemDiscountResponse{
		ID:           d.ID,
		ItemID:       d.ItemID,
		DiscountType: d.DiscountType,
		Value:        d.Value,
		StartsAt:     d.StartsAt,
		EndsAt:       d.EndsAt,
		IsActive:     d.IsActive,
	}
}

type UpdateItemDiscount struct {
	bun.BaseModel `bun:"table:products.item_discounts"`

	ID           uuid.UUID  `bun:"id,pk,type:uuid"`
	DiscountType string     `bun:"discount_type,notnull"`
	Value        float64    `bun:"value,notnull"`
	StartsAt     *time.Time `bun:"starts_at,nullzero"`
	EndsAt       *time.Time `bun:"ends_at,nullzero"`
	IsActive     bool       `bun:"is_active"`
	UpdatedAt    time.Time  `bun:"updated_at"`
}

func (u *UpdateItemDiscount) BuildToModel(discount dtos.UpdateItemDiscount, discountID uuid.UUID) {
	u.ID = discountID
	u.DiscountType = discount.DiscountType
	u.Value = discount.Value
	u.StartsAt = discount.StartsAt
	u.EndsAt = discount.EndsAt
	u.IsActive = discount.IsActive
	u.UpdatedAt = time.Now()
}
