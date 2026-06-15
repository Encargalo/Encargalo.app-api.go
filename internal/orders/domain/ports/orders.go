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
	SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string, requesterID uuid.UUID, isShop bool) (models.Order, error)
	ConfirmPickup(ctx context.Context, orderID uuid.UUID, code string) (models.Order, error)
	ConfirmDelivery(ctx context.Context, orderID uuid.UUID, code string) error
}

type OrdersRepo interface {
	CreateOrder(ctx context.Context, order *models.Order) (*models.Order, error)
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error)
	SearchItemsByID(ctx context.Context, ItemsID []uuid.UUID) (items.Items, error)
	SearchShopByID(ctx context.Context, shopID uuid.UUID) (*models.ShopOrder, error)
	SearchAdditionsByID(ctx context.Context, additionID []uuid.UUID) ([]models.Addition, error)
	SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) error
	SavePickupCode(ctx context.Context, orderID uuid.UUID, code string) error
	SaveDeliveryCode(ctx context.Context, orderID uuid.UUID, code string) error
	MarkPickupCodeUsed(ctx context.Context, orderID uuid.UUID) error
	MarkDeliveryCodeUsed(ctx context.Context, orderID uuid.UUID) error
	AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error
	SearchActiveOrdersByRiderID(ctx context.Context, riderID uuid.UUID) ([]models.Order, error)
}

// OrderRiderBroadcaster is implemented by the riders module to receive order lifecycle events.
// Defined here so orders/app has no import dependency on riders/domain/ports.
type OrderRiderBroadcaster interface {
	BroadcastNewOrder(order models.Order)
	BroadcastOrderUpdate(order models.Order)
}
