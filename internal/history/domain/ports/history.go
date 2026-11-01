package ports

import (
	"context"

	"Encargalo.app-api.go/internal/history/domain/dtos"
	ordersDtos "Encargalo.app-api.go/internal/orders/domain/dtos"
	shipmentsDtos "Encargalo.app-api.go/internal/shipments/domain/dtos"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
)

type HistoryApp interface {
	GetHistory(ctx context.Context, customerID uuid.UUID) (dtos.HistoryResponse, error)
}

// OrdersHistoryProvider es el subconjunto de ordersPorts.OrdersApp que history
// necesita — mismo patrón que ridersPorts.TripsStore/ShipmentsStore: un
// puerto angosto definido por quien consume, no por quien implementa. Lo
// satisface ordersPorts.OrdersApp sin wrapper, por estructura.
type OrdersHistoryProvider interface {
	GetOrderHistory(ctx context.Context, customerID uuid.UUID) (ordersDtos.OrderHistoryResponse, error)
}

// TripsHistoryProvider es el equivalente de OrdersHistoryProvider para
// viajes. Lo satisface tripsPorts.TripsApp sin wrapper.
type TripsHistoryProvider interface {
	GetTripHistory(ctx context.Context, customerID uuid.UUID) (tripsDtos.TripHistoryResponse, error)
}

// ShipmentsHistoryProvider es el equivalente de OrdersHistoryProvider para
// envíos. Lo satisface shipmentsPorts.ShipmentsApp sin wrapper.
type ShipmentsHistoryProvider interface {
	GetShipmentHistory(ctx context.Context, customerID uuid.UUID) (shipmentsDtos.ShipmentHistoryResponse, error)
}
