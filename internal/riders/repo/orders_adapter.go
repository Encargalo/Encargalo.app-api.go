package repo

import (
	"context"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"github.com/google/uuid"
)

type ridersRepoAdapter struct {
	orders ordersPorts.OrdersRepo
}

func NewRidersRepo(orders ordersPorts.OrdersRepo) ridersPorts.RidersRepo {
	return &ridersRepoAdapter{orders}
}

func (r *ridersRepoAdapter) AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error {
	return r.orders.AssignRider(ctx, orderID, riderID)
}

func (r *ridersRepoAdapter) SearchActiveOrdersByRiderID(ctx context.Context, riderID uuid.UUID) ([]ordersModels.Order, error) {
	return r.orders.SearchActiveOrdersByRiderID(ctx, riderID)
}

func (r *ridersRepoAdapter) SearchAvailableOrders(ctx context.Context) ([]ordersModels.Order, error) {
	return r.orders.SearchAvailableOrders(ctx)
}

func (r *ridersRepoAdapter) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*ordersModels.Order, error) {
	return r.orders.SearchOrdersByID(ctx, orderID)
}

func (r *ridersRepoAdapter) SearchShopByID(ctx context.Context, shopID uuid.UUID) (*ordersModels.ShopOrder, error) {
	return r.orders.SearchShopByID(ctx, shopID)
}
