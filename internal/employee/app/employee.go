package app

import (
	"context"

	"Encargalo.app-api.go/internal/employee/domain/dto"
	"Encargalo.app-api.go/internal/employee/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	"github.com/google/uuid"
)

type employeeApp struct {
	repo ports.EmployeeRepo
}

func New(repo ports.EmployeeRepo) ports.EmployeeApp {
	return &employeeApp{
		repo,
	}
}

func (e *employeeApp) SearchEmployeeByPhone(ctx context.Context, phone string) (models.Account, error) {
	return e.repo.SearchEmployeeByPhone(ctx, phone)
}

func (e *employeeApp) SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (dto.EmployeeResponse, error) {
	return e.repo.SearchEmployeeByID(ctx, employeeID)
}
