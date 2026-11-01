package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
)

type CustomerShipmentHub struct {
	mu     sync.RWMutex
	conns  map[uuid.UUID]ports.CustomerConn
	repo   ports.ShipmentsRepo
	riders ports.RiderPublicInfoLookup
}

func NewCustomerShipmentHub(repo ports.ShipmentsRepo, riders ports.RiderPublicInfoLookup) *CustomerShipmentHub {
	return &CustomerShipmentHub{
		conns:  make(map[uuid.UUID]ports.CustomerConn),
		repo:   repo,
		riders: riders,
	}
}

func (h *CustomerShipmentHub) Register(ctx context.Context, customerID uuid.UUID, conn ports.CustomerConn) error {
	h.mu.Lock()
	h.conns[customerID] = conn
	h.mu.Unlock()

	if h.repo == nil {
		return nil
	}
	shipments, err := h.repo.ListActiveShipmentsByCustomer(ctx, customerID)
	if err != nil {
		return err
	}
	for i := range shipments {
		h.sendShipment(ctx, conn, &shipments[i])
	}
	return nil
}

func (h *CustomerShipmentHub) Unregister(customerID uuid.UUID, conn ports.CustomerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if current, ok := h.conns[customerID]; ok && current == conn {
		delete(h.conns, customerID)
	}
}

func (h *CustomerShipmentHub) NotifyShipment(ctx context.Context, shipment models.Shipment) {
	h.mu.RLock()
	conn := h.conns[shipment.CustomerID]
	h.mu.RUnlock()
	if conn == nil {
		return
	}
	h.sendShipment(ctx, conn, &shipment)
}

func (h *CustomerShipmentHub) NotifyRiderLocation(_ context.Context, customerID, shipmentID uuid.UUID, lat, lng float64) {
	h.mu.RLock()
	conn := h.conns[customerID]
	h.mu.RUnlock()
	if conn == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type":        "rider_location",
		"shipment_id": shipmentID,
		"latitude":    lat,
		"longitude":   lng,
	})
	if err != nil {
		slog.Error("customer shipment hub marshal rider_location", "error", err)
		return
	}
	conn.Send(payload)
}

func (h *CustomerShipmentHub) sendShipment(ctx context.Context, conn ports.CustomerConn, shipment *models.Shipment) {
	resp := shipment.ToCustomerDTO()
	if shipment.RiderID != nil && h.riders != nil {
		info, err := h.riders.GetByRiderID(ctx, *shipment.RiderID)
		if err != nil {
			slog.Error("customer shipment hub rider lookup", "shipmentID", shipment.ID, "error", err)
		} else {
			resp.Rider = &info
		}
	}
	payload, err := json.Marshal(map[string]any{
		"type":     "shipment_update",
		"shipment": resp,
	})
	if err != nil {
		slog.Error("customer shipment hub marshal shipment_update", "error", err)
		return
	}
	conn.Send(payload)
}
