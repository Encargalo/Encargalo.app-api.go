package models

import (
	"Encargalo.app-api.go/internal/employee/domain/dto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type EmployeeShop struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID     uuid.UUID `bun:"id,pk,type:uuid"`
	Name   string    `bun:"name"`
	Opened bool      `bun:"opened"`
}

func (e *EmployeeShop) ToDomainDTO() dto.ShopResponse {
	return dto.ShopResponse{
		ID:     e.ID,
		Name:   e.Name,
		Opened: e.Opened,
	}
}
