package app

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

type appManagerOrder struct {
	mu                  sync.RWMutex
	shopConnections     map[uuid.UUID][]ports.ManagerConn
	customerConnections map[uuid.UUID][]ports.ManagerConn
	svc                 ports.OrdersApp
}

type orderBroadcast struct {
	Type  string       `json:"type"`
	Order models.Order `json:"order"`
}

func NewManagerOrderApp(svc ports.OrdersApp) ports.AppManagerOrder {
	return &appManagerOrder{
		shopConnections:     make(map[uuid.UUID][]ports.ManagerConn),
		customerConnections: make(map[uuid.UUID][]ports.ManagerConn),
		svc:                 svc,
	}
}

func (s *appManagerOrder) Register(
	ctx context.Context,
	connectionID uuid.UUID,
	conn ports.ManagerConn,
	isShop bool,
) error {
	s.mu.Lock()
	if isShop {
		s.shopConnections[connectionID] = append(s.shopConnections[connectionID], conn)
	} else {
		s.customerConnections[connectionID] = append(s.customerConnections[connectionID], conn)
	}
	s.mu.Unlock()

	orders, err := s.svc.SearchActiveOrdersByCriteria(ctx, connectionID, isShop)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	role := models.ClientRoleCustomer
	if isShop {
		role = models.ClientRoleShop
	}
	for _, order := range orders {
		data, _ := json.Marshal(orderBroadcast{Type: "order_update", Order: stripCodes(order, role)})
		conn.Send(data)
	}

	return nil
}

func (s *appManagerOrder) Unregister(
	connectionID uuid.UUID,
	conn ports.ManagerConn,
	isShop bool,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var connections map[uuid.UUID][]ports.ManagerConn
	if isShop {
		connections = s.shopConnections
	} else {
		connections = s.customerConnections
	}

	if list, ok := connections[connectionID]; ok {
		filtered := list[:0]
		for _, c := range list {
			if c != conn {
				filtered = append(filtered, c)
			}
		}

		if len(filtered) == 0 {
			delete(connections, connectionID)
		} else {
			connections[connectionID] = filtered
		}
	}

	conn.Close()
}

func (s *appManagerOrder) BroadcastOrder(order models.Order) error {
	shopData, err := json.Marshal(orderBroadcast{Type: "order_update", Order: stripCodes(order, models.ClientRoleShop)})
	if err != nil {
		return err
	}

	customerData, err := json.Marshal(orderBroadcast{Type: "order_update", Order: stripCodes(order, models.ClientRoleCustomer)})
	if err != nil {
		return err
	}

	s.mu.RLock()
	shopClients := s.shopConnections[order.ShopID]
	customerClients := s.customerConnections[order.CustomerID]
	s.mu.RUnlock()

	for _, c := range shopClients {
		c.Send(shopData)
	}
	for _, c := range customerClients {
		c.Send(customerData)
	}

	return nil
}

// stripCodes filtra los códigos de confirmación según el rol del destinatario.
// Shop → ninguno; Customer → solo delivery_code; Rider → solo pickup_code.
func stripCodes(order models.Order, role models.ClientRole) models.Order {
	switch role {
	case models.ClientRoleShop:
		order.PickupCode = nil
		order.DeliveryCode = nil
	case models.ClientRoleCustomer:
		order.PickupCode = nil
	case models.ClientRoleRider:
		order.DeliveryCode = nil
	}
	return order
}
