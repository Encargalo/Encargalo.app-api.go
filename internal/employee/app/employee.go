package app

import (
	"context"

	"Encargalo.app-api.go/internal/employee/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
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
