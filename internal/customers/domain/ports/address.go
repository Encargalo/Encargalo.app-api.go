package ports

import (
	"context"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"github.com/google/uuid"
)

type CustomersAddressApp interface {
	RegisterAddress(ctx context.Context, customerID uuid.UUID, address dto.Address) error
	SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (dto.Addresses, error)
	DeleteAddress(ctx context.Context, customer_id, address_id uuid.UUID) error
}

type CustomersAddressRepo interface {
	RegisterAddress(ctx context.Context, address models.Address) error
	SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (dto.Addresses, error)
	DeleteAddress(ctx context.Context, customer_id, address_id uuid.UUID) error
}
