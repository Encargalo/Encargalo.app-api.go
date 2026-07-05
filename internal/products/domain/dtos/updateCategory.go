package dtos

import (
	"context"

	"github.com/google/uuid"
)

type UpdateCategory struct {
	Name            string           `json:"name" validate:"required" example:"Desayuno"`
	MacroCategoryID *uuid.UUID       `json:"macro_category_id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	Schedules       *[]ScheduleInput `json:"schedules" validate:"omitempty,dive"`
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
