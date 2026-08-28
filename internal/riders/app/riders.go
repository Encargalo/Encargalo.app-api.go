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
	"Encargalo.app-api.go/internal/pkg/geo"
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
	riderMode       map[uuid.UUID]riderModels.RiderMode
	riderLocation   map[uuid.UUID]riderModels.RiderLocation
	repo            ridersPorts.RidersRepo
	stream          ridersPorts.RidersStream
	bcv             bcvPorts.BcvApp
}

func New(repo ridersPorts.RidersRepo, stream ridersPorts.RidersStream, bcv bcvPorts.BcvApp) ridersPorts.RidersApp {
	return &appRidersManager{
		availableRiders: make(map[uuid.UUID]ridersPorts.RiderConn),
		assignedOrders:  make(map[uuid.UUID][]uuid.UUID),
		riderMode:       make(map[uuid.UUID]riderModels.RiderMode),
		riderLocation:   make(map[uuid.UUID]riderModels.RiderLocation),
		repo:            repo,
		stream:          stream,
		bcv:             bcv,
	}
}

func (r *appRidersManager) Register(ctx context.Context, riderID uuid.UUID, conn ridersPorts.RiderConn, mode riderModels.RiderMode) error {
	if mode != riderModels.RiderModeTrips {
		mode = riderModels.RiderModeOrders
	}

	r.mu.Lock()
	r.availableRiders[riderID] = conn
	r.riderMode[riderID] = mode
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
		deliveryfee.ApplyBsToOrder(ctx, r.bcv, &order)
		data, _ := json.Marshal(riderBroadcast{
			Type:     "order_update",
			Order:    riderModels.NewRiderOrder(order, true),
			Shop:     r.shopForOrder(ctx, order),
			Customer: customerLocation(order, true),
		})
		conn.Send(data)
	}

	availableOrders, err := r.repo.SearchAvailableOrders(ctx)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	for _, order := range availableOrders {
		deliveryfee.ApplyBsToOrder(ctx, r.bcv, &order)
		data, _ := json.Marshal(riderBroadcast{
			Type:     "new_order",
			Order:    riderModels.NewRiderOrder(order, false),
			Shop:     r.shopForOrder(ctx, order),
			Customer: customerLocation(order, false),
		})
		conn.Send(data)
	}

	return nil
}

func (r *appRidersManager) Unregister(riderID uuid.UUID, conn ridersPorts.RiderConn) {
	r.mu.Lock()
	if stored, ok := r.availableRiders[riderID]; ok && stored == conn {
		delete(r.availableRiders, riderID)
		delete(r.riderMode, riderID)
		delete(r.riderLocation, riderID)
	}
	r.mu.Unlock()

	conn.Close()
}

func (r *appRidersManager) UpdateLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64) error {
	cell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	if err != nil {
		return errcustom.ErrInvalidCoordinates
	}

	r.mu.Lock()
	r.riderLocation[riderID] = riderModels.RiderLocation{
		Latitude:  lat,
		Longitude: lng,
		H3Cell:    cell,
		UpdatedAt: time.Now(),
	}
	r.mu.Unlock()

	return nil
}

func (r *appRidersManager) SnapshotTripsRiders() []riderModels.RiderLocationEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]riderModels.RiderLocationEntry, 0, len(r.riderLocation))
	for riderID, loc := range r.riderLocation {
		if r.riderMode[riderID] != riderModels.RiderModeTrips {
			continue
		}
		out = append(out, riderModels.RiderLocationEntry{RiderID: riderID, Location: loc})
	}
	return out
}

func (r *appRidersManager) BroadcastNewOrder(ctx context.Context, order ordersModels.Order) {
	data, err := json.Marshal(riderBroadcast{
		Type:     "new_order",
		Order:    riderModels.NewRiderOrder(order, false),
		Shop:     r.shopForOrder(ctx, order),
		Customer: customerLocation(order, false),
	})
	if err != nil {
		slog.Error("error marshaling new_order broadcast", "error", err)
		return
	}

	r.mu.RLock()
	clients := make([]ridersPorts.RiderConn, 0, len(r.availableRiders))
	for riderID, c := range r.availableRiders {
		if r.riderMode[riderID] == riderModels.RiderModeTrips {
			continue
		}
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

func (r *appRidersManager) BroadcastOrderUpdate(ctx context.Context, order ordersModels.Order) {
	data, err := json.Marshal(riderBroadcast{
		Type:     "order_update",
		Order:    riderModels.NewRiderOrder(order, true),
		Shop:     r.shopForOrder(ctx, order),
		Customer: customerLocation(order, true),
	})
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
	accepted, _ := json.Marshal(riderBroadcast{
		Type:     "order_accepted",
		Order:    riderModels.NewRiderOrder(*order, true),
		Shop:     r.shopForOrder(ctx, *order),
		Customer: customerLocation(*order, true),
	})
	conn.Send(accepted)

	return nil
}

// shopForOrder fetches the shop's public info (name, logo, address, etc.) to
// embed alongside the order. Errors are logged, not propagated: a missing
// shop must not block delivering the order event to riders.
func (r *appRidersManager) shopForOrder(ctx context.Context, order ordersModels.Order) *ordersModels.ShopOrder {
	shop, err := r.repo.SearchShopByID(ctx, order.ShopID)
	if err != nil {
		slog.Error("error fetching shop for rider broadcast", "shopID", order.ShopID, "error", err)
		return nil
	}
	return shop
}

// customerLocation builds the customer info sent to riders. Before a rider
// accepts (new_order, broadcast to every available rider), includeContact
// must be false so no PII leaks. Once a rider has accepted the order
// (order_accepted, order_update — both scoped to that one rider), pass true
// to also include the customer's name, last name and phone.
func customerLocation(order ordersModels.Order, includeContact bool) *riderModels.CustomerLocation {
	var details struct {
		Address struct {
			Instructions string `json:"instructions"`
		} `json:"address"`
		Customer struct {
			Name     string `json:"name"`
			LastName string `json:"last_name"`
			Phone    string `json:"phone"`
		} `json:"customer"`
	}
	_ = json.Unmarshal(order.Details, &details)

	location := &riderModels.CustomerLocation{
		Address:      order.Address,
		Latitude:     order.Latitude,
		Longitude:    order.Longitude,
		Instructions: details.Address.Instructions,
	}

	if includeContact {
		location.Name = details.Customer.Name
		location.LastName = details.Customer.LastName
		location.Phone = details.Customer.Phone
	}

	return location
}

type riderBroadcast struct {
	Type     string                        `json:"type"`
	Order    riderModels.RiderOrder        `json:"order"`
	Shop     *ordersModels.ShopOrder       `json:"shop,omitempty"`
	Customer *riderModels.CustomerLocation `json:"customer,omitempty"`
}
