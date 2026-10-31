package dtos

import "fmt"

type ScheduleInput struct {
	DayOfWeek int16  `json:"day_of_week" validate:"min=0,max=6"`
	Open      string `json:"open" validate:"required,datetime=15:04:05" example:"08:00:00"`
	Close     string `json:"close" validate:"required,datetime=15:04:05" example:"20:00:00"`
	Enabled   *bool  `json:"enabled" example:"true"`
}

func (s ScheduleInput) EnabledOrDefault() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

type ScheduleResponse struct {
	DayOfWeek int16  `json:"day_of_week"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	Enabled   bool   `json:"enabled"`
}

type ShopScheduleRequest struct {
	Schedules []ScheduleInput `json:"schedules" validate:"dive"`
}

func (r *ShopScheduleRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return err
	}
	for _, s := range r.Schedules {
		if s.Open == s.Close {
			return fmt.Errorf("open y close no pueden ser iguales")
		}
	}
	return nil
}
