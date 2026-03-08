package ports

import (
	"context"

	"Encargalo.app-api.go/internal/employee/domain/dto"
	"github.com/google/uuid"
)

type EmployeeApp interface {
	SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (dto.EmployeeResponse, error)
}

type EmployeeRepo interface {
	SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (dto.EmployeeResponse, error)
}
