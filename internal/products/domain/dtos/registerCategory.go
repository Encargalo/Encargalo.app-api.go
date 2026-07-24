package dtos

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ScheduleInput struct {
	StartTime string `json:"start_time" validate:"required,datetime=15:04:05" example:"06:00:00"`
	EndTime   string `json:"end_time" validate:"required,datetime=15:04:05" example:"11:00:00"`
	Enabled   *bool  `json:"enabled" example:"true"`
}

func (s ScheduleInput) EnabledOrDefault() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

type FlavorInput struct {
	Name        string `json:"name" validate:"required" example:"Jamón"`
	Description string `json:"description" example:"Con jamón ahumado"`
}

type RegisterCategory struct {
	ShopID          uuid.UUID       `json:"shop_id" swaggerignore:"true"`
	Name            string          `json:"name" validate:"required" example:"Desayuno"`
	MacroCategoryID uuid.UUID       `json:"macro_category_id" validate:"required" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	SelectorType    string          `json:"selector_type" validate:"omitempty,oneof=none single_select multi_select" example:"single_select"`
	Flavors         []FlavorInput   `json:"flavors" validate:"dive"`
	Schedules       []ScheduleInput `json:"schedules" validate:"dive"`
}

func (r *RegisterCategory) Validate() error {
	_ = conform.Struct(context.Background(), r)
	if err := validate.Struct(r); err != nil {
		return err
	}
	if len(r.Flavors) > 0 {
		selector := r.SelectorType
		if selector == "" {
			selector = "none"
		}
		if selector == "none" {
			return fmt.Errorf("selector_type debe ser single_select o multi_select cuando se envían flavors")
		}
	}
	return validateSchedules(r.Schedules)
}

func validateSchedules(schedules []ScheduleInput) error {
	for _, s := range schedules {
		if s.StartTime == s.EndTime {
			return fmt.Errorf("start_time y end_time no pueden ser iguales")
		}
	}
	return nil
}
