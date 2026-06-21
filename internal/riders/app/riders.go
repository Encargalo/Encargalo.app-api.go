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
	financeRepo     ridersPorts.FinanceRepo
	bcv             bcvPorts.BcvApp
}

func New(repo ridersPorts.RidersRepo, stream ridersPorts.RidersStream, financeRepo ridersPorts.FinanceRepo, bcv bcvPorts.BcvApp) ridersPorts.RidersApp {
	return &appRidersManager{
		availableRiders: make(map[uuid.UUID]ridersPorts.RiderConn),
		assignedOrders:  make(map[uuid.UUID][]uuid.UUID),
		repo:            repo,
		stream:          stream,
		financeRepo:     financeRepo,
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
		data, _ := json.Marshal(riderBroadcast{Type: "order_update", Order: stripPickupOnly(order)})
		conn.Send(data)
	}

	availableOrders, err := r.repo.SearchAvailableOrders(ctx)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	for _, order := range availableOrders {
		stripped := order
		stripped.PickupCode = nil
		stripped.DeliveryCode = nil
		deliveryfee.ApplyBsToOrder(ctx, r.bcv, &stripped)
		earning := RiderEarning(stripped.DeliveryFee)
		msg := riderBroadcast{Type: "new_order", Order: stripped, RiderEarnings: &earning}
		if isCashPayment(stripped.MethodPayment) {
			total := stripped.TotalPrice
			msg.CollectAmount = &total
		}
		data, _ := json.Marshal(msg)
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
	stripped := order
	stripped.PickupCode = nil
	stripped.DeliveryCode = nil

	earning := RiderEarning(order.DeliveryFee)
	msg := riderBroadcast{Type: "new_order", Order: stripped, RiderEarnings: &earning}
	if isCashPayment(order.MethodPayment) {
		total := order.TotalPrice
		msg.CollectAmount = &total
	}

	data, err := json.Marshal(msg)
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
	data, err := json.Marshal(riderBroadcast{Type: "order_update", Order: stripPickupOnly(order)})
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

	balance, err := r.financeRepo.GetBalance(ctx, riderID, entityTypeRider)
	if err != nil {
		return err
	}
	if getZone(balance) == riderModels.ZoneRisk && isCashPayment(order.MethodPayment) {
		return errcustom.ErrRiderInRiskZone
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
	earning := RiderEarning(order.DeliveryFee)
	acceptedMsg := riderBroadcast{Type: "order_accepted", Order: stripPickupOnly(*order), RiderEarnings: &earning}
	if isCashPayment(order.MethodPayment) {
		total := order.TotalPrice
		acceptedMsg.CollectAmount = &total
	}
	accepted, _ := json.Marshal(acceptedMsg)
	conn.Send(accepted)

	return nil
}

func stripPickupOnly(order ordersModels.Order) ordersModels.Order {
	order.DeliveryCode = nil
	return order
}

type riderBroadcast struct {
	Type          string             `json:"type"`
	Order         ordersModels.Order `json:"order"`
	RiderEarnings *float64           `json:"rider_earnings,omitempty"`
	CollectAmount *float64           `json:"collect_amount,omitempty"`
}
