package dtos

import (
	"time"

	ordersDtos "Encargalo.app-api.go/internal/orders/domain/dtos"
	shipmentsDtos "Encargalo.app-api.go/internal/shipments/domain/dtos"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
)

type HistoryItemType string

const (
	HistoryItemOrder    HistoryItemType = "order"
	HistoryItemTrip     HistoryItemType = "trip"
	HistoryItemShipment HistoryItemType = "shipment"
)

// HistoryItem envuelve el item de su dominio sin aplanarlo: cada uno tiene su
// propio desglose (productos y pago en order, rider y códigos en trip/shipment)
// y forzarlos a un shape común perdería esos campos. El cliente distingue por
// type cuál de los tres leer; los otros dos vienen omitidos.
type HistoryItem struct {
	Type      HistoryItemType `json:"type"`
	CreatedAt time.Time       `json:"created_at"`

	Order    *ordersDtos.OrderHistoryItem    `json:"order,omitempty"`
	Trip     *tripsDtos.TripResponse         `json:"trip,omitempty"`
	Shipment *shipmentsDtos.ShipmentResponse `json:"shipment,omitempty"`
}

// HistoryResponse va de más reciente a más antiguo, igual que los tres
// historiales individuales.
type HistoryResponse []HistoryItem
