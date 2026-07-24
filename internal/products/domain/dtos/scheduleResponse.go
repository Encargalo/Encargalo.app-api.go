package dtos

import "github.com/google/uuid"

type ScheduleResponse struct {
	ID        uuid.UUID `json:"id"`
	DayOfWeek *int16    `json:"day_of_week,omitempty"`
	StartTime string    `json:"start_time" example:"18:00:00"`
	EndTime   string    `json:"end_time" example:"02:00:00"`
	Enabled   bool      `json:"enabled"`
}
