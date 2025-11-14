package ports

import (
	"context"

	"Encargalo.app-api.go/internal/employee/domain/models"
	"github.com/google/uuid"
)

type EmployeeApp interface {
	SearchEmployeeByPhone(ctx context.Context, phone string) (models.Account, error)
	SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (models.Account, error)
}

type EmployeeRepo interface {
	SearchEmployeeByPhone(ctx context.Context, phone string) (models.Account, error)
	SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (models.Account, error)
}
