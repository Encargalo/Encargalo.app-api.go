package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"sync"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/pkg/geo"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

const orderAcceptTimeout = 10 * time.Minute

const (
	tripOfferCandidateCount = 3
	tripSearchMaxRings      = 3
	tripAcceptTimeout       = 2 * time.Minute
)

type appRidersManager struct {
	mu              sync.RWMutex
	availableRiders map[uuid.UUID]ridersPorts.RiderConn
	assignedOrders  map[uuid.UUID][]uuid.UUID // riderID → []orderID
	riderMode       map[uuid.UUID]riderModels.RiderMode
	riderLocation   map[uuid.UUID]riderModels.RiderLocation
	tripOffers      map[uuid.UUID]map[uuid.UUID]struct{}
	tripTimers      map[uuid.UUID]*time.Timer
	completingTrips map[uuid.UUID]struct{}
	repo            ridersPorts.RidersRepo
	stream          ridersPorts.RidersStream
	bcv             bcvPorts.BcvApp
	tripsRepo       ridersPorts.TripsStore
	tripEarnings    ridersPorts.TripEarningRecorder
}

func New(repo ridersPorts.RidersRepo, stream ridersPorts.RidersStream, bcv bcvPorts.BcvApp, tripsRepo ridersPorts.TripsStore, tripEarnings ridersPorts.TripEarningRecorder) ridersPorts.RidersApp {
	return &appRidersManager{
		availableRiders: make(map[uuid.UUID]ridersPorts.RiderConn),
		assignedOrders:  make(map[uuid.UUID][]uuid.UUID),
		riderMode:       make(map[uuid.UUID]riderModels.RiderMode),
		riderLocation:   make(map[uuid.UUID]riderModels.RiderLocation),
		tripOffers:      make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripTimers:      make(map[uuid.UUID]*time.Timer),
		completingTrips: make(map[uuid.UUID]struct{}),
		repo:            repo,
		stream:          stream,
		bcv:             bcv,
		tripsRepo:       tripsRepo,
		tripEarnings:    tripEarnings,
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

func (r *appRidersManager) OfferTrip(ctx context.Context, trip tripsModels.Trip) {
	candidates := r.nearestTripsRiders(trip)
	if len(candidates) == 0 {
		r.expireTripIfStillSearching(trip.ID)
		return
	}

	data, err := json.Marshal(tripBroadcast{Type: "trip_offer", Trip: trip.ToDTO()})
	if err != nil {
		slog.Error("error marshaling trip_offer broadcast", "error", err)
		return
	}

	offered := make(map[uuid.UUID]struct{}, len(candidates))
	for _, c := range candidates {
		offered[c.RiderID] = struct{}{}
	}
	r.mu.Lock()
	r.tripOffers[trip.ID] = offered
	r.mu.Unlock()

	r.mu.RLock()
	for _, c := range candidates {
		if conn, ok := r.availableRiders[c.RiderID]; ok {
			conn.Send(data)
		}
	}
	r.mu.RUnlock()

	timer := time.AfterFunc(tripAcceptTimeout, func() {
		r.expireTripIfStillSearching(trip.ID)
	})
	r.mu.Lock()
	r.tripTimers[trip.ID] = timer
	r.mu.Unlock()
}

func (r *appRidersManager) expireTripIfStillSearching(tripID uuid.UUID) {
	ctx := context.Background()
	_, err := r.tripsRepo.UpdateTripStatus(ctx, tripID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver)
	if err != nil {
		slog.Error("error expiring trip to no_driver", "tripID", tripID, "error", err)
	}

	r.mu.Lock()
	delete(r.tripOffers, tripID)
	delete(r.tripTimers, tripID)
	r.mu.Unlock()
}

func (r *appRidersManager) wasOfferedTrip(tripID, riderID uuid.UUID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tripOffers[tripID][riderID]
	return ok
}

// beginCompleting marca el trip como "completándose" si no lo estaba ya,
// de forma atómica. Devuelve false si otra llamada a completeTrip para el
// mismo trip ya está en curso (ver completeTrip).
func (r *appRidersManager) beginCompleting(tripID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, inFlight := r.completingTrips[tripID]; inFlight {
		return false
	}
	r.completingTrips[tripID] = struct{}{}
	return true
}

func (r *appRidersManager) endCompleting(tripID uuid.UUID) {
	r.mu.Lock()
	delete(r.completingTrips, tripID)
	r.mu.Unlock()
}

func (r *appRidersManager) nearestTripsRiders(trip tripsModels.Trip) []riderModels.RiderLocationEntry {
	if trip.PickupH3Cell == nil {
		return nil
	}

	byCell := make(map[string][]riderModels.RiderLocationEntry)
	for _, e := range r.SnapshotTripsRiders() {
		byCell[e.Location.H3Cell] = append(byCell[e.Location.H3Cell], e)
	}

	for k := 0; k <= tripSearchMaxRings; k++ {
		cells, err := geo.GridDisk(*trip.PickupH3Cell, k)
		if err != nil {
			slog.Error("error expanding H3 rings for trip matching", "tripID", trip.ID, "error", err)
			return nil
		}

		matched := make([]riderModels.RiderLocationEntry, 0, tripOfferCandidateCount)
		for _, cell := range cells {
			matched = append(matched, byCell[cell]...)
		}

		if len(matched) < tripOfferCandidateCount && k < tripSearchMaxRings {
			continue
		}

		return nearestByDistance(trip, matched)
	}

	return nil
}

func nearestByDistance(trip tripsModels.Trip, candidates []riderModels.RiderLocationEntry) []riderModels.RiderLocationEntry {
	type scored struct {
		entry riderModels.RiderLocationEntry
		km    float64
	}

	ranked := make([]scored, len(candidates))
	for i, e := range candidates {
		ranked[i] = scored{
			entry: e,
			km:    geo.HaversineKm(trip.PickupLatitude, trip.PickupLongitude, e.Location.Latitude, e.Location.Longitude),
		}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].km < ranked[j].km })

	if len(ranked) > tripOfferCandidateCount {
		ranked = ranked[:tripOfferCandidateCount]
	}

	out := make([]riderModels.RiderLocationEntry, len(ranked))
	for i, s := range ranked {
		out[i] = s.entry
	}
	return out
}

func (r *appRidersManager) AcceptTrip(ctx context.Context, riderID, tripID uuid.UUID, conn ridersPorts.RiderConn) error {
	trip, err := r.tripsRepo.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.Status != tripsModels.TripStatusSearching {
		return errcustom.ErrInvalidTransition
	}
	if !r.wasOfferedTrip(tripID, riderID) {
		return errcustom.ErrInvalidTransition
	}

	ok, err := r.tripsRepo.AssignTripRider(ctx, tripID, riderID)
	if err != nil {
		return err
	}
	if !ok {
		msg, _ := json.Marshal(map[string]string{"type": "trip_taken", "trip_id": tripID.String()})
		conn.Send(msg)
		return nil
	}

	r.mu.Lock()
	delete(r.tripOffers, tripID)
	if timer, ok := r.tripTimers[tripID]; ok {
		timer.Stop()
		delete(r.tripTimers, tripID)
	}
	r.mu.Unlock()

	acceptedAt := time.Now()
	trip.RiderID = &riderID
	trip.Status = tripsModels.TripStatusAccepted
	trip.AcceptedAt = &acceptedAt
	accepted, _ := json.Marshal(tripBroadcast{Type: "trip_accepted", Trip: trip.ToDTO()})
	conn.Send(accepted)

	return nil
}

// AdvanceTrip mueve el trip asignado al siguiente estado. Solo el rider
// asignado puede avanzarlo; in_progress→completed se resuelve en
// completeTrip porque además calcula la tarifa y acredita la ganancia.
// Vive en riders/app (no en trips/app) a propósito, igual que AcceptTrip:
// ver "Trips & Riders" en CLAUDE.md para el porqué de este split.
func (r *appRidersManager) AdvanceTrip(ctx context.Context, riderID, tripID uuid.UUID, conn ridersPorts.RiderConn) error {
	trip, ok, err := r.tripsRepo.AdvanceTripForward(ctx, tripID, riderID)
	if err != nil {
		return err
	}
	if ok {
		data, _ := json.Marshal(tripBroadcast{Type: "trip_status", Trip: trip.ToDTO()})
		conn.Send(data)
		return nil
	}

	// No aplicó la transición atómica; leemos para distinguir in_progress (completar) de un caso inválido.
	trip, err = r.tripsRepo.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.RiderID == nil || *trip.RiderID != riderID {
		return errcustom.ErrInvalidTransition
	}
	if trip.Status != tripsModels.TripStatusInProgress {
		return errcustom.ErrInvalidTransition
	}
	return r.completeTrip(ctx, trip, conn)
}

// completeTrip calcula la tarifa por distancia vial pickup→dropoff, persiste
// la transición in_progress→completed junto con esa tarifa de forma atómica,
// y acredita la ganancia del rider. Un fallo al acreditar (ej. tasa BCV no
// disponible) no revierte la finalización del trip: se loguea y el rider
// puede reclamarlo por soporte, igual que RecordRideEarning en orders.
//
// beginCompleting/endCompleting evitan que un spam de advance_trip sobre el
// mismo trip dispare múltiples llamadas reales a Google Maps (tripFare) en
// paralelo mientras la primera aún no confirma en DB.
func (r *appRidersManager) completeTrip(ctx context.Context, trip *tripsModels.Trip, conn ridersPorts.RiderConn) error {
	if !r.beginCompleting(trip.ID) {
		return errcustom.ErrInvalidTransition
	}
	defer r.endCompleting(trip.ID)

	fare, err := r.tripFare(ctx, trip)
	if err != nil {
		return err
	}

	updated, err := r.tripsRepo.CompleteTrip(ctx, trip.ID, fare)
	if err != nil {
		return err
	}
	if !updated {
		return errcustom.ErrInvalidTransition
	}

	completedAt := time.Now()
	trip.Status = tripsModels.TripStatusCompleted
	trip.Fare = &fare
	trip.CompletedAt = &completedAt
	data, _ := json.Marshal(tripBroadcast{Type: "trip_status", Trip: trip.ToDTO()})
	conn.Send(data)

	if err := r.tripEarnings.RecordTripEarning(ctx, *trip.RiderID, trip.ID, fare, r.bcvRate(ctx)); err != nil {
		slog.Error("error recording trip earning", "tripID", trip.ID, "riderID", *trip.RiderID, "error", err)
	}

	return nil
}

// tripFare calcula la tarifa por distancia vial pickup→dropoff. Fuera de
// producción usa un placeholder fijo para no gastar cuota de Google Maps,
// mismo patrón que priceOrderDraft en orders/app.
func (r *appRidersManager) tripFare(ctx context.Context, trip *tripsModels.Trip) (float64, error) {
	if !config.Get().IsProduction() {
		return deliveryfee.DevPlaceholderFee, nil
	}
	return deliveryfee.Calculate(ctx, trip.PickupLatitude, trip.PickupLongitude, trip.DropoffLatitude, trip.DropoffLongitude)
}

func (r *appRidersManager) bcvRate(ctx context.Context) float64 {
	rate, err := r.bcv.GetRate(ctx)
	if err != nil || rate == nil {
		return 0
	}
	return rate.Rate
}

type tripBroadcast struct {
	Type string                 `json:"type"`
	Trip tripsDtos.TripResponse `json:"trip"`
}
