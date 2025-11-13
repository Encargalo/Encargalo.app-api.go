package ports

import (
	"context"

	"Encargalo.app-api.go/internal/employee/domain/models"
)

type EmployeeApp interface {
	SearchEmployeeByPhone(ctx context.Context, phone string) (models.Account, error)
}

type EmployeeRepo interface {
	SearchEmployeeByPhone(ctx context.Context, phone string) (models.Account, error)
}
