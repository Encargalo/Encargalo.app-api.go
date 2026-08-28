package ports

import (
	"context"

	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
)

// RiderConn abstracts the WebSocket transport so the app layer has no infrastructure dependency.
type RiderConn interface {
	Send(msg []byte)
	Close()
}

// RidersApp defines the operations exposed to the riders HTTP handler.
// Broadcast methods are excluded here; they are accessed via
// orders/domain/ports.OrderRiderBroadcaster by the orders domain.
type RidersApp interface {
	Register(ctx context.Context, riderID uuid.UUID, conn RiderConn, mode riderModels.RiderMode) error
	Unregister(riderID uuid.UUID, conn RiderConn)
	AcceptOrder(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, conn RiderConn) error
	UpdateLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64) error
	SnapshotTripsRiders() []riderModels.RiderLocationEntry
}

type RidersStream interface {
	PublisherEvent(ctx context.Context, data any) error
}
