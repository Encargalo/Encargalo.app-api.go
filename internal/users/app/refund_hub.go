package app

import (
	"context"
	"encoding/json"
	"sync"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

type StaffRefundHubImpl struct {
	mu          sync.RWMutex
	connections map[uuid.UUID][]ports.RefundWSConn
	finance     ports.CustomerFinanceApp
}

func NewStaffRefundHub() *StaffRefundHubImpl {
	return &StaffRefundHubImpl{
		connections: make(map[uuid.UUID][]ports.RefundWSConn),
	}
}

func (h *StaffRefundHubImpl) SetFinanceApp(finance ports.CustomerFinanceApp) {
	h.finance = finance
}

func (h *StaffRefundHubImpl) Register(ctx context.Context, staffID uuid.UUID, conn ports.RefundWSConn) error {
	h.mu.Lock()
	h.connections[staffID] = append(h.connections[staffID], conn)
	h.mu.Unlock()

	if h.finance == nil {
		return nil
	}
	items, err := h.finance.ListActiveForStaff(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		data, _ := json.Marshal(item)
		conn.Send(data)
	}
	return nil
}

func (h *StaffRefundHubImpl) Unregister(staffID uuid.UUID, conn ports.RefundWSConn) {
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

func (h *StaffRefundHubImpl) Broadcast(req dto.RefundRequestBroadcast) error {
	data, err := json.Marshal(req)
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

var _ ports.StaffRefundHub = (*StaffRefundHubImpl)(nil)
