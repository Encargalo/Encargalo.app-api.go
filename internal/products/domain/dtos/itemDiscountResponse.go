package dtos

import (
	"time"

	"github.com/google/uuid"
)

type ItemDiscountResponse struct {
	ID           uuid.UUID  `json:"id"`
	ItemID       uuid.UUID  `json:"item_id"`
	DiscountType string     `json:"discount_type"`
	Value        float64    `json:"value"`
	StartsAt     *time.Time `json:"starts_at,omitempty"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	IsActive     bool       `json:"is_active"`
}
