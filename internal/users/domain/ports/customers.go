package ports

import (
	"context"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type CustomersApp interface {
	RegisterCustomer(ctx context.Context, customer dto.RegisterCustomer) error
	SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Account, error)
	UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer dto.UpdateCustomer) error
	UpdatePassword(ctx context.Context, customer_id uuid.UUID, pass dto.UpdatePassword) error
}

type CustomersRepo interface {
	RegisterCustomer(ctx context.Context, customer *models.Account) error
	SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Account, error)
	SearchCustomerByPhoneAndNotIDEquals(ctx context.Context, customer_id uuid.UUID, phone string) (*models.Account, error)
	UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer *models.Account) error
	UpdatePassword(ctx context.Context, customer_id uuid.UUID, customer *models.Account) error
}
