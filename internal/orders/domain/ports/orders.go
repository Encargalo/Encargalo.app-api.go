package ports

import (
	"context"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	items "Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
)

type OrdersApp interface {
	CreateOrder(ctx context.Context, order *dtos.CreateOrder) (*models.Order, error)
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error)
	SearchOrderByBatchIsOpen(ctx context.Context, shopID uuid.UUID) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) (models.Order, error)
}

type OrdersRepo interface {
	CreateOrder(ctx context.Context, order *models.Order) error
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error)
	SearchItemsByID(ctx context.Context, ItemsID []uuid.UUID) (items.Items, error)
	SearchAdditionsByID(ctx context.Context, additionID []uuid.UUID) ([]models.Addition, error)
	SearchOrderByBatchIsOpen(ctx context.Context, shopID uuid.UUID) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) error
}
