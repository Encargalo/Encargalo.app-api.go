package app

import (
	"context"
	"sort"

	"Encargalo.app-api.go/internal/history/domain/dtos"
	"Encargalo.app-api.go/internal/history/domain/ports"
	"github.com/google/uuid"
)

type historyApp struct {
	orders    ports.OrdersHistoryProvider
	trips     ports.TripsHistoryProvider
	shipments ports.ShipmentsHistoryProvider
}

func NewHistoryApp(orders ports.OrdersHistoryProvider, trips ports.TripsHistoryProvider, shipments ports.ShipmentsHistoryProvider) ports.HistoryApp {
	return &historyApp{orders: orders, trips: trips, shipments: shipments}
}

// GetHistory combina los tres historiales del customer en un solo listado
// ordenado de más reciente a más antiguo. No aplana los items: cada uno trae
// su type y solo el campo (order/trip/shipment) que le corresponde, con el
// mismo shape que su endpoint de historial individual.
func (a *historyApp) GetHistory(ctx context.Context, customerID uuid.UUID) (dtos.HistoryResponse, error) {
	orders, err := a.orders.GetOrderHistory(ctx, customerID)
	if err != nil {
		return nil, err
	}
	trips, err := a.trips.GetTripHistory(ctx, customerID)
	if err != nil {
		return nil, err
	}
	shipments, err := a.shipments.GetShipmentHistory(ctx, customerID)
	if err != nil {
		return nil, err
	}

	items := make(dtos.HistoryResponse, 0, len(orders)+len(trips.Trips)+len(shipments.Shipments))
	for i := range orders {
		order := orders[i]
		items = append(items, dtos.HistoryItem{Type: dtos.HistoryItemOrder, CreatedAt: order.CreatedAt, Order: &order})
	}
	for i := range trips.Trips {
		trip := trips.Trips[i]
		items = append(items, dtos.HistoryItem{Type: dtos.HistoryItemTrip, CreatedAt: trip.RequestedAt, Trip: &trip})
	}
	for i := range shipments.Shipments {
		shipment := shipments.Shipments[i]
		items = append(items, dtos.HistoryItem{Type: dtos.HistoryItemShipment, CreatedAt: shipment.RequestedAt, Shipment: &shipment})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return items, nil
}
