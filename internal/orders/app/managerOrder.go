package app

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

type appManagerOrder struct {
	mu                  sync.RWMutex
	shopConnections     map[uuid.UUID][]ports.ManagerConn
	customerConnections map[uuid.UUID][]ports.ManagerConn
	svc                 ports.OrdersApp
	bcv                 bcvPorts.BcvApp
}

const shopCommissionRate = 0.10

func NewManagerOrderApp(svc ports.OrdersApp, bcv bcvPorts.BcvApp) ports.AppManagerOrder {
	return &appManagerOrder{
		shopConnections:     make(map[uuid.UUID][]ports.ManagerConn),
		customerConnections: make(map[uuid.UUID][]ports.ManagerConn),
		svc:                 svc,
		bcv:                 bcv,
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

	for _, order := range orders {
		deliveryfee.ApplyBsToOrder(ctx, s.bcv, &order)
		data, _ := json.Marshal(buildBroadcast("order_update", order, isShop))
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

func (s *appManagerOrder) BroadcastShopStatus(shopID uuid.UUID, open bool) error {
	data, err := json.Marshal(map[string]any{"type": "shop_status", "opened": open})
	if err != nil {
		return err
	}

	s.mu.RLock()
	clients := s.shopConnections[shopID]
	s.mu.RUnlock()

	for _, c := range clients {
		c.Send(data)
	}
	return nil
}

func (s *appManagerOrder) BroadcastOrder(order models.Order) error {
	shopData, err := json.Marshal(buildBroadcast("order_update", order, true))
	if err != nil {
		return err
	}

	customerData, err := json.Marshal(buildBroadcast("order_update", order, false))
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

func (s *appManagerOrder) BroadcastOrderToCustomer(order models.Order) error {
	customerData, err := json.Marshal(buildBroadcast("order_update", order, false))
	if err != nil {
		return err
	}

	s.mu.RLock()
	customerClients := s.customerConnections[order.CustomerID]
	s.mu.RUnlock()

	for _, c := range customerClients {
		c.Send(customerData)
	}

	return nil
}

func (s *appManagerOrder) NotifyCustomer(customerID uuid.UUID, msg []byte) error {
	s.mu.RLock()
	customerClients := s.customerConnections[customerID]
	s.mu.RUnlock()

	for _, c := range customerClients {
		c.Send(msg)
	}
	return nil
}

func buildBroadcast(msgType string, order models.Order, isShop bool) models.OrderBroadcast {
	b := models.OrderBroadcast{Type: msgType}
	if isShop {
		// Debe reflejar exactamente la misma cuenta que shopFinanceApp.
		// RecordSaleCommission (internal/shops/app/finance.go) usa al liquidar,
		// para que lo que ve el negocio en tiempo real coincida con lo que
		// realmente se le acredita: durante la promo día 1 la plataforma
		// repone el 50% subsidiado (PromoSubsidyAmount) y durante la ventana
		// de comisión 0% (IsZeroCommission) el negocio se queda con el 100%
		// en vez del 90% habitual.
		base := order.TotalPrice - order.DeliveryFee
		if order.IsLaunchPromo {
			base += order.PromoSubsidyAmount
		}
		rate := shopCommissionRate
		if order.IsZeroCommission {
			rate = 0
		}
		commission := base * rate
		net := base - commission
		b.Order = stripCodes(order, models.ClientRoleShop)
		b.SaleAmount = &base
		b.Commission = &commission
		b.NetAmount = &net
	} else {
		b.Order = stripCodes(order, models.ClientRoleCustomer)
	}
	return b
}

func stripCodes(order models.Order, role models.ClientRole) models.Order {
	switch role {
	case models.ClientRoleShop:
		order.PickupCode = nil
		order.DeliveryCode = nil
		order.TotalPrice = order.TotalPrice - order.DeliveryFee
		order.DeliveryFee = 0
	case models.ClientRoleCustomer:
		order.PickupCode = nil
	case models.ClientRoleRider:
		order.DeliveryCode = nil
	}
	return order
}
