package ports

import (
	"context"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type UsersAddressApp interface {
	RegisterAddress(ctx context.Context, customerID uuid.UUID, address dto.Address) error
	SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (models.Addresses, error)
	DeleteAddress(ctx context.Context, customer_id uuid.UUID, address_id string) error
}

type UsersAddressRepo interface {
	RegisterAddress(ctx context.Context, address models.Address) error
	SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (models.Addresses, error)
	DeleteAddress(ctx context.Context, customer_id uuid.UUID, address_id string) error
}
