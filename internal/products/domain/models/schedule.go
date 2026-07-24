package models

import (
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Schedules []Schedule

type Schedule struct {
	bun.BaseModel `bun:"table:products.schedules" swaggerignore:"true"`

	ID         uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	CategoryID *uuid.UUID `bun:"category_id" json:"category_id,omitempty"`
	ItemID     *uuid.UUID `bun:"item_id" json:"item_id,omitempty"`
	DayOfWeek  *int16     `bun:"day_of_week" json:"day_of_week,omitempty"`
	StartTime  string     `bun:"start_time" json:"start_time"`
	EndTime    string     `bun:"end_time" json:"end_time"`
	Enabled    bool       `bun:"enabled,default:true" json:"enabled"`
	CreatedAt  time.Time  `bun:"created_at,default:now()" json:"-"`
}

func (s *Schedule) ToDomainDTO() dtos.ScheduleResponse {
	return dtos.ScheduleResponse{
		ID:        s.ID,
		DayOfWeek: s.DayOfWeek,
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		Enabled:   s.Enabled,
	}
}

func (s *Schedules) ToDomainDTO() []dtos.ScheduleResponse {
	schedules := make([]dtos.ScheduleResponse, 0, len(*s))

	for _, v := range *s {
		schedules = append(schedules, v.ToDomainDTO())
	}

	return schedules
}
