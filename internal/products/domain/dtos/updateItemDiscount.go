package dtos

import (
	"context"
	"errors"
	"time"
)

type UpdateItemDiscount struct {
	DiscountType string     `json:"discount_type" validate:"required,oneof=percentage fixed" example:"percentage"`
	Value        float64    `json:"value" validate:"required,gt=0" example:"20"`
	StartsAt     *time.Time `json:"starts_at,omitempty" example:"2026-07-11T00:00:00Z"`
	EndsAt       *time.Time `json:"ends_at,omitempty" example:"2026-07-18T00:00:00Z"`
	IsActive     bool       `json:"is_active" example:"true"`
}

func (u *UpdateItemDiscount) Validate() error {
	_ = conform.Struct(context.Background(), u)
	if err := validate.Struct(u); err != nil {
		return err
	}
	if u.StartsAt != nil && u.EndsAt != nil && !u.StartsAt.Before(*u.EndsAt) {
		return errors.New("starts_at debe ser anterior a ends_at")
	}
	return nil
}
