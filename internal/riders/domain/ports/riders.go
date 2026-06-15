package ports

import (
	"context"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
)

// RiderConn abstracts the WebSocket transport so the app layer has no infrastructure dependency.
type RiderConn interface {
	Send(msg []byte)
	Close()
}

type RidersApp interface {
	Register(ctx context.Context, riderID uuid.UUID, conn RiderConn) error
	Unregister(riderID uuid.UUID, conn RiderConn)
	BroadcastNewOrder(order ordersModels.Order)
	BroadcastOrderUpdate(order ordersModels.Order)
	AcceptOrder(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, conn RiderConn) error
}

type RidersRepo interface {
	AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error
	SearchActiveOrdersByRiderID(ctx context.Context, riderID uuid.UUID) ([]ordersModels.Order, error)
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*ordersModels.Order, error)
}

type RidersStream interface {
	PublisherEvent(ctx context.Context, data any) error
}
