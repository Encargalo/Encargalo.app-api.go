package app

import (
	"context"
	"encoding/json"
	"sync"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"github.com/google/uuid"
)

// StaffReviewHubImpl is the concrete hub; dig needs the concrete type to call SetReviewsApp after PaymentReviewsApp is built.
type StaffReviewHubImpl struct {
	mu          sync.RWMutex
	connections map[uuid.UUID][]ports.ManagerConn
	reviews     ports.PaymentReviewsApp
}

func NewStaffReviewHub() *StaffReviewHubImpl {
	return &StaffReviewHubImpl{
		connections: make(map[uuid.UUID][]ports.ManagerConn),
	}
}

func (h *StaffReviewHubImpl) SetReviewsApp(reviews ports.PaymentReviewsApp) {
	h.reviews = reviews
}

func (h *StaffReviewHubImpl) Register(ctx context.Context, staffID uuid.UUID, conn ports.ManagerConn) error {
	h.mu.Lock()
	h.connections[staffID] = append(h.connections[staffID], conn)
	h.mu.Unlock()

	if h.reviews == nil {
		return nil
	}

	items, err := h.reviews.ListActiveForStaff(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		data, _ := json.Marshal(item)
		conn.Send(data)
	}
	return nil
}

func (h *StaffReviewHubImpl) Unregister(staffID uuid.UUID, conn ports.ManagerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	list, ok := h.connections[staffID]
	if !ok {
		conn.Close()
		return
	}

	filtered := list[:0]
	for _, c := range list {
		if c != conn {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		delete(h.connections, staffID)
	} else {
		h.connections[staffID] = filtered
	}
	conn.Close()
}

func (h *StaffReviewHubImpl) BroadcastReview(review models.PaymentReviewBroadcast) error {
	data, err := json.Marshal(review)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.connections {
		for _, c := range clients {
			c.Send(data)
		}
	}
	return nil
}

var _ ports.StaffReviewHub = (*StaffReviewHubImpl)(nil)
