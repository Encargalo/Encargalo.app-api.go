package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/pkg/money"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

const orderAcceptTimeout = 10 * time.Minute

type appRidersManager struct {
	mu              sync.RWMutex
	availableRiders map[uuid.UUID]ridersPorts.RiderConn
	assignedOrders  map[uuid.UUID][]uuid.UUID // riderID → []orderID
	repo            ridersPorts.RidersRepo
	stream          ridersPorts.RidersStream
	bcv             bcvPorts.BcvApp
}

func New(repo ridersPorts.RidersRepo, stream ridersPorts.RidersStream, bcv bcvPorts.BcvApp) ridersPorts.RidersApp {
	return &appRidersManager{
		availableRiders: make(map[uuid.UUID]ridersPorts.RiderConn),
		assignedOrders:  make(map[uuid.UUID][]uuid.UUID),
		repo:            repo,
		stream:          stream,
		bcv:             bcv,
	}
}

func (r *appRidersManager) Register(ctx context.Context, riderID uuid.UUID, conn ridersPorts.RiderConn) error {
	r.mu.Lock()
	r.availableRiders[riderID] = conn
	r.mu.Unlock()

	activeOrders, err := r.repo.SearchActiveOrdersByRiderID(ctx, riderID)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	// Collect order IDs under lock, then send outside lock to avoid holding mutex while writing to channel.
	r.mu.Lock()
	orderIDs := make([]uuid.UUID, 0, len(activeOrders))
	for _, order := range activeOrders {
		orderIDs = append(orderIDs, order.ID)
	}
	r.assignedOrders[riderID] = append(r.assignedOrders[riderID], orderIDs...)
	r.mu.Unlock()

	for _, order := range activeOrders {
		data, _ := json.Marshal(riderBroadcast{Type: "order_update", Order: stripForRider(order)})
		conn.Send(data)
	}

	availableOrders, err := r.repo.SearchAvailableOrders(ctx)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	for _, order := range availableOrders {
		stripped := stripForRider(order)
		deliveryfee.ApplyBsToOrder(ctx, r.bcv, &stripped)
		data, _ := json.Marshal(riderBroadcast{Type: "new_order", Order: stripped})
		conn.Send(data)
	}

	return nil
}

func (r *appRidersManager) Unregister(riderID uuid.UUID, conn ridersPorts.RiderConn) {
	r.mu.Lock()
	if stored, ok := r.availableRiders[riderID]; ok && stored == conn {
		delete(r.availableRiders, riderID)
	}
	r.mu.Unlock()

	conn.Close()
}

func (r *appRidersManager) BroadcastNewOrder(order ordersModels.Order) {
	stripped := stripForRider(order)
	data, err := json.Marshal(riderBroadcast{Type: "new_order", Order: stripped})
	if err != nil {
		slog.Error("error marshaling new_order broadcast", "error", err)
		return
	}

	r.mu.RLock()
	clients := make([]ridersPorts.RiderConn, 0, len(r.availableRiders))
	for _, c := range r.availableRiders {
		clients = append(clients, c)
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.Send(data)
	}

	orderID := order.ID
	go func() {
		time.Sleep(orderAcceptTimeout)
		ctx := context.Background()
		o, err := r.repo.SearchOrdersByID(ctx, orderID)
		if err != nil || o.RiderID != nil {
			return
		}
		event := ordersModels.DataEventOrderNoRider{}
		event.BuildFromOrder(*o)
		if err := r.stream.PublisherEvent(ctx, event); err != nil {
			slog.Error("error publishing order_no_rider event", "orderID", orderID, "error", err)
		}
	}()
}

func (r *appRidersManager) BroadcastOrderUpdate(order ordersModels.Order) {
	data, err := json.Marshal(riderBroadcast{Type: "order_update", Order: stripForRider(order)})
	if err != nil {
		slog.Error("error marshaling order_update for rider", "error", err)
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for riderID, orderIDs := range r.assignedOrders {
		if slices.Contains(orderIDs, order.ID) {
			if c, ok := r.availableRiders[riderID]; ok {
				c.Send(data)
			}
		}
	}
}

func (r *appRidersManager) AcceptOrder(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, conn ridersPorts.RiderConn) error {
	// Validate existence and state before attempting atomic assignment (option B).
	order, err := r.repo.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != string(ordersModels.StatusInPreparation) && order.Status != string(ordersModels.StatusReady) {
		return errcustom.ErrInvalidTransition
	}

	if err := r.repo.AssignRider(ctx, orderID, riderID); err != nil {
		if errors.Is(err, errcustom.ErrOrderAlreadyAssigned) {
			msg, _ := json.Marshal(map[string]string{"type": "order_taken", "order_id": orderID.String()})
			conn.Send(msg)
			return nil
		}
		return err
	}

	// Reload to get updated rider_id / rider_accepted_at.
	order, err = r.repo.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.assignedOrders[riderID] = append(r.assignedOrders[riderID], orderID)
	r.mu.Unlock()

	deliveryfee.ApplyBsToOrder(ctx, r.bcv, order)
	accepted, _ := json.Marshal(riderBroadcast{Type: "order_accepted", Order: stripForRider(*order)})
	conn.Send(accepted)

	return nil
}

func stripForRider(order ordersModels.Order) ordersModels.Order {
	order.DeliveryCode = nil
	order.TotalPrice = 0
	order.TotalPriceBs = 0
	order.Details = nil
	order.DeliveryFee = order.DeliveryFee * riderModels.RiderShare
	order.DeliveryFeeBs = money.RoundBs(order.DeliveryFeeBs * riderModels.RiderShare)
	return order
}

type riderBroadcast struct {
	Type  string             `json:"type"`
	Order ordersModels.Order `json:"order"`
}
