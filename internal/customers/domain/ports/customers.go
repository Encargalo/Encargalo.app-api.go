package ports

import (
	"context"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"github.com/google/uuid"
)

type CustomersApp interface {
	RegisterCustomer(ctx context.Context, customer dto.RegisterCustomer) (uuid.UUID, error)
	SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Accounts, error)
	UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer dto.UpdateCustomer) error
	UpdatePassword(ctx context.Context, customer_id uuid.UUID, pass dto.UpdatePassword) error
}

type CustomersRepo interface {
	RegisterCustomer(ctx context.Context, customer *models.Accounts) (*models.Accounts, error)
	SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Accounts, error)
	SearchCustomerByPhoneAndNotIDEquals(ctx context.Context, customer_id uuid.UUID, phone string) (*models.Accounts, error)
	UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer *models.Accounts) error
	UpdatePassword(ctx context.Context, customer_id uuid.UUID, customer *models.Accounts) error
}
