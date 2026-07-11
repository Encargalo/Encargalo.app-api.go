package dtos

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type RegisterItemDiscount struct {
	ItemID       uuid.UUID  `json:"item_id" validate:"required" example:"7e441237-a818-42d2-bb54-9b8747198305"`
	ShopID       uuid.UUID  `json:"shop_id" swaggerignore:"true"`
	DiscountType string     `json:"discount_type" validate:"required,oneof=percentage fixed" example:"percentage"`
	Value        float64    `json:"value" validate:"required,gt=0" example:"20"`
	StartsAt     *time.Time `json:"starts_at,omitempty" example:"2026-07-11T00:00:00Z"`
	EndsAt       *time.Time `json:"ends_at,omitempty" example:"2026-07-18T00:00:00Z"`
}

func (r *RegisterItemDiscount) Validate() error {
	_ = conform.Struct(context.Background(), r)
	if err := validate.Struct(r); err != nil {
		return err
	}
	if r.StartsAt != nil && r.EndsAt != nil && !r.StartsAt.Before(*r.EndsAt) {
		return errors.New("starts_at debe ser anterior a ends_at")
	}
	return nil
}
