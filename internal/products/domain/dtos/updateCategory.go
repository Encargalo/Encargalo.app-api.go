package dtos

import (
	"context"

	"github.com/google/uuid"
)

type UpdateCategory struct {
	Name               string           `json:"name" validate:"required" example:"Desayuno"`
	MacroCategoryID    uuid.UUID        `json:"macro_category_id" validate:"required" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	SelectorType       *string          `json:"selector_type" validate:"omitempty,oneof=none single_select multi_select" example:"single_select"`
	AvailabilityStatus *string          `json:"availability_status" validate:"omitempty,oneof=active hidden out_of_stock" example:"active"`
	Schedules          *[]ScheduleInput `json:"schedules" validate:"omitempty,dive"`
}

func (u *UpdateCategory) Validate() error {
	_ = conform.Struct(context.Background(), u)
	if err := validate.Struct(u); err != nil {
		return err
	}
	if u.Schedules == nil {
		return nil
	}
	return validateSchedules(*u.Schedules)
}
