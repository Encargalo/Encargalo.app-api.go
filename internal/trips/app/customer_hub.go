package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type CustomerTripHub struct {
	mu     sync.RWMutex
	conns  map[uuid.UUID]ports.CustomerConn
	repo   ports.TripsRepo
	riders ports.RiderPublicInfoLookup
}

func NewCustomerTripHub(repo ports.TripsRepo, riders ports.RiderPublicInfoLookup) *CustomerTripHub {
	return &CustomerTripHub{
		conns:  make(map[uuid.UUID]ports.CustomerConn),
		repo:   repo,
		riders: riders,
	}
}

func (h *CustomerTripHub) Register(ctx context.Context, customerID uuid.UUID, conn ports.CustomerConn) error {
	h.mu.Lock()
	h.conns[customerID] = conn
	h.mu.Unlock()

	if h.repo == nil {
		return nil
	}
	trips, err := h.repo.ListActiveTripsByCustomer(ctx, customerID)
	if err != nil {
		return err
	}
	for i := range trips {
		h.sendTrip(ctx, conn, &trips[i])
	}
	return nil
}

func (h *CustomerTripHub) Unregister(customerID uuid.UUID, conn ports.CustomerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if current, ok := h.conns[customerID]; ok && current == conn {
		delete(h.conns, customerID)
	}
}

func (h *CustomerTripHub) NotifyTrip(ctx context.Context, trip models.Trip) {
	h.mu.RLock()
	conn := h.conns[trip.CustomerID]
	h.mu.RUnlock()
	if conn == nil {
		return
	}
	h.sendTrip(ctx, conn, &trip)
}

func (h *CustomerTripHub) NotifyRiderLocation(_ context.Context, customerID, tripID uuid.UUID, lat, lng float64) {
	h.mu.RLock()
	conn := h.conns[customerID]
	h.mu.RUnlock()
	if conn == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type":      "rider_location",
		"trip_id":   tripID,
		"latitude":  lat,
		"longitude": lng,
	})
	if err != nil {
		slog.Error("customer hub marshal rider_location", "error", err)
		return
	}
	conn.Send(payload)
}

func (h *CustomerTripHub) sendTrip(ctx context.Context, conn ports.CustomerConn, trip *models.Trip) {
	resp := trip.ToCustomerDTO()
	if trip.RiderID != nil && h.riders != nil {
		info, err := h.riders.GetByRiderID(ctx, *trip.RiderID)
		if err != nil {
			slog.Error("customer hub rider lookup", "tripID", trip.ID, "error", err)
		} else {
			resp.Rider = &info
		}
	}
	payload, err := json.Marshal(map[string]any{
		"type": "trip_update",
		"trip": resp,
	})
	if err != nil {
		slog.Error("customer hub marshal trip_update", "error", err)
		return
	}
	conn.Send(payload)
}
