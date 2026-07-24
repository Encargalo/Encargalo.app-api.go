package ports

import (
	"context"

	// ordersModels is a deliberate cross-domain dependency: this port adapts
	// order data for use within the riders domain. It is intentionally isolated
	// to this file to keep the boundary explicit and auditable.
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
)

type RidersRepo interface {
	AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error
	SearchActiveOrdersByRiderID(ctx context.Context, riderID uuid.UUID) ([]ordersModels.Order, error)
	SearchAvailableOrders(ctx context.Context) ([]ordersModels.Order, error)
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*ordersModels.Order, error)
	SearchShopByID(ctx context.Context, shopID uuid.UUID) (*ordersModels.ShopOrder, error)
}
