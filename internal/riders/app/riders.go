package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsApp "Encargalo.app-api.go/internal/trips/app"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	tripsPorts "Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

// tripForwardStatus cubre accepted→arriving→in_progress; in_progress→completed se maneja en completeTrip.
var tripForwardStatus = map[tripsModels.TripStatus]tripsModels.TripStatus{
	tripsModels.TripStatusAccepted: tripsModels.TripStatusArriving,
	tripsModels.TripStatusArriving: tripsModels.TripStatusInProgress,
}

const orderAcceptTimeout = 10 * time.Minute

const (
	tripOfferCandidateCount = 3
	// tripSearchMaxRings es el tope de anillos H3 (res 8) de la primera ronda.
	// En 7 el disco cubre ~125 km²: garantiza 5 km en toda dirección y alcanza
	// 7 km en las esquinas. Cada ronda posterior suma un anillo.
	tripSearchMaxRings = 7
	// tripAcceptTimeout es lo que dura una ronda con candidatos ofertados.
	tripAcceptTimeout = 10 * time.Minute
	// tripEmptyRoundTimeout es lo que dura una ronda sin riders elegibles, antes
	// de pasar a la siguiente (puede conectarse alguien mientras tanto).
	tripEmptyRoundTimeout = 5 * time.Minute
	// tripOfferMaxRounds es el tope de rondas, con o sin candidatos, antes de no_driver.
	tripOfferMaxRounds = 4
)

type appRidersManager struct {
	mu                 sync.RWMutex
	availableRiders    map[uuid.UUID]ridersPorts.RiderConn
	assignedOrders     map[uuid.UUID][]uuid.UUID // riderID → []orderID
	orderCustomer      map[uuid.UUID]uuid.UUID   // orderID → customerID, para avisarle la ubicación del rider
	riderMode          map[uuid.UUID]riderModels.RiderMode
	riderLocation      map[uuid.UUID]riderModels.RiderLocation
	riderVehicle       map[uuid.UUID]string
	ridersByCell       map[string]map[uuid.UUID]struct{}
	tripOffers         map[uuid.UUID]map[uuid.UUID]struct{}
	tripAlreadyOffered map[uuid.UUID]map[uuid.UUID]struct{}
	tripOfferRounds    map[uuid.UUID]int
	tripOfferTrip      map[uuid.UUID]tripsModels.Trip
	tripExcluded       map[uuid.UUID]map[uuid.UUID]struct{}
	tripTimers         map[uuid.UUID]*time.Timer
	// tripWithdrawn marca viajes que el cliente ya canceló. Una ronda que ya
	// pasó el chequeo de status no puede volver a publicar la oferta.
	tripWithdrawn   map[uuid.UUID]struct{}
	completingTrips map[uuid.UUID]struct{}
	// acceptingRiders evita que un rider acepte dos viajes a la vez antes de quedar en riderTrips.
	acceptingRiders map[uuid.UUID]struct{}
	repo            ridersPorts.RidersRepo
	stream          ridersPorts.RidersStream
	bcv             bcvPorts.BcvApp
	tripsRepo       ridersPorts.TripsStore
	tripEarnings    ridersPorts.TripEarningRecorder
	customers       tripsPorts.CustomerTripNotifier
	balance         paymentsPorts.CustomerBalanceCreditor
	passengers      ridersPorts.PassengerLookup
	vehicles        ridersPorts.VehicleLookup
	riderTrips      map[uuid.UUID]riderAssignment
	orderNotifier   ridersPorts.OrderCustomerNotifier
}

type riderAssignment struct {
	tripID     uuid.UUID
	customerID uuid.UUID
}

func New(repo ridersPorts.RidersRepo, stream ridersPorts.RidersStream, bcv bcvPorts.BcvApp, tripsRepo ridersPorts.TripsStore, tripEarnings ridersPorts.TripEarningRecorder, customers tripsPorts.CustomerTripNotifier, balance paymentsPorts.CustomerBalanceCreditor, passengers ridersPorts.PassengerLookup, vehicles ridersPorts.VehicleLookup) ridersPorts.RidersApp {
	return &appRidersManager{
		availableRiders:    make(map[uuid.UUID]ridersPorts.RiderConn),
		assignedOrders:     make(map[uuid.UUID][]uuid.UUID),
		orderCustomer:      make(map[uuid.UUID]uuid.UUID),
		riderMode:          make(map[uuid.UUID]riderModels.RiderMode),
		riderLocation:      make(map[uuid.UUID]riderModels.RiderLocation),
		riderVehicle:       make(map[uuid.UUID]string),
		ridersByCell:       make(map[string]map[uuid.UUID]struct{}),
		tripOffers:         make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripAlreadyOffered: make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripOfferRounds:    make(map[uuid.UUID]int),
		tripOfferTrip:      make(map[uuid.UUID]tripsModels.Trip),
		tripExcluded:       make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripTimers:         make(map[uuid.UUID]*time.Timer),
		tripWithdrawn:      make(map[uuid.UUID]struct{}),
		completingTrips:    make(map[uuid.UUID]struct{}),
		acceptingRiders:    make(map[uuid.UUID]struct{}),
		repo:               repo,
		stream:             stream,
		bcv:                bcv,
		tripsRepo:          tripsRepo,
		tripEarnings:       tripEarnings,
		customers:          customers,
		balance:            balance,
		passengers:         passengers,
		vehicles:           vehicles,
		riderTrips:         make(map[uuid.UUID]riderAssignment),
	}
}

// riderTripOfferDTO es el viaje como lo ve el rider: fare_usd/fare_bs son su
// ganancia (RiderShare), igual que delivery_fee en los pedidos. collect_usd/
// collect_bs siguen siendo el total que cobra en efectivo. Bs siempre con la
// tasa congelada del viaje.
func riderTripOfferDTO(trip *tripsModels.Trip) tripsDtos.TripResponse {
	frozen := *trip
	frozen.ApplyFrozenBsFare()
	resp := frozen.ToDTO()
	resp.DiscountUSD = 0
	if trip.Fare == nil {
		return resp
	}
	// El pago del rider sale de la tarifa completa: el descuento de primer viaje lo absorbe la plataforma.
	gross := trip.GrossFareUSD()
	usd := money.RoundUSD(riderModels.RiderPayUSD(gross))
	resp.FareUSD = &usd
	if trip.BcvRate != nil && *trip.BcvRate > 0 {
		bs := money.RoundBs(riderModels.RiderPayUSD(money.RoundBs(gross * *trip.BcvRate)))
		resp.FareBs = &bs
	}
	return resp
}

// riderTripDTO es la vista del viaje para el rider asignado. El nombre y el
// apellido del pasajero salen siempre; el teléfono entra solo en accepted,
// arriving e in_progress.
func (r *appRidersManager) riderTripDTO(ctx context.Context, trip *tripsModels.Trip) tripsDtos.TripResponse {
	resp := riderTripOfferDTO(trip)
	if r.passengers == nil {
		return resp
	}
	contact, err := r.passengers.GetByCustomerID(ctx, trip.CustomerID)
	if err != nil {
		slog.Error("error loading passenger contact for assigned rider", "tripID", trip.ID, "error", err)
		return resp
	}
	if !exposesPassengerContact(trip.Status) {
		contact.Phone = ""
	}
	resp.Customer = &contact
	return resp
}

// tripOfferWithPassenger es la oferta de viaje con nombre y apellido del
// pasajero, antes de que algún rider acepte. Nunca incluye el teléfono.
func (r *appRidersManager) tripOfferWithPassenger(ctx context.Context, trip *tripsModels.Trip) tripsDtos.TripResponse {
	resp := riderTripOfferDTO(trip)
	if r.passengers == nil {
		return resp
	}
	contact, err := r.passengers.GetByCustomerID(ctx, trip.CustomerID)
	if err != nil {
		slog.Error("error loading passenger name for trip offer", "tripID", trip.ID, "error", err)
		return resp
	}
	resp.Customer = &tripsDtos.PassengerContact{Name: contact.Name, LastName: contact.LastName}
	return resp
}

func exposesPassengerContact(status tripsModels.TripStatus) bool {
	switch status {
	case tripsModels.TripStatusAccepted, tripsModels.TripStatusArriving, tripsModels.TripStatusInProgress:
		return true
	default:
		return false
	}
}

func (r *appRidersManager) Register(ctx context.Context, riderID uuid.UUID, conn ridersPorts.RiderConn, mode riderModels.RiderMode) error {
	if mode != riderModels.RiderModeTrips {
		mode = riderModels.RiderModeOrders
	}

	r.mu.Lock()
	r.availableRiders[riderID] = conn
	r.riderMode[riderID] = mode
	if loc, ok := r.riderLocation[riderID]; ok {
		r.reindexRiderLocked(riderID, loc.H3Cell, loc.H3Cell, mode == riderModels.RiderModeTrips)
	}
	r.mu.Unlock()

	vehicle := r.cacheRiderVehicle(ctx, riderID)

	activeOrders, err := r.repo.SearchActiveOrdersByRiderID(ctx, riderID)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	// Collect order IDs under lock, then send outside lock to avoid holding mutex while writing to channel.
	r.mu.Lock()
	orderIDs := make([]uuid.UUID, 0, len(activeOrders))
	for _, order := range activeOrders {
		orderIDs = append(orderIDs, order.ID)
		r.orderCustomer[order.ID] = order.CustomerID
	}
	r.assignedOrders[riderID] = append(r.assignedOrders[riderID], orderIDs...)
	r.mu.Unlock()

	for _, order := range activeOrders {
		deliveryfee.ApplyBsToOrder(ctx, r.bcv, &order)
		data, _ := json.Marshal(r.orderBroadcast(ctx, "order_update", order, true))
		conn.Send(data)
	}

	availableOrders, err := r.repo.SearchAvailableOrders(ctx)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return err
	}

	// Los activos van sin filtrar: ya están asignados a este rider. Acá se filtran los que podría tomar.
	for _, order := range availableOrders {
		deliveryfee.ApplyBsToOrder(ctx, r.bcv, &order)
		msg := r.orderBroadcast(ctx, "new_order", order, false)
		if msg.Shop != nil && !riderModels.VehicleCanCarryShopType(vehicle, msg.Shop.Type) {
			continue
		}
		data, _ := json.Marshal(msg)
		conn.Send(data)
	}

	return nil
}

// cacheRiderVehicle trae el vehículo a memoria: el matching lo consulta con el lock tomado.
func (r *appRidersManager) cacheRiderVehicle(ctx context.Context, riderID uuid.UUID) string {
	if r.vehicles == nil {
		return ""
	}
	profile, err := r.vehicles.GetByUserID(ctx, riderID)
	if err != nil {
		slog.Error("error leyendo el vehículo del rider", "riderID", riderID, "error", err)
		return ""
	}
	if profile == nil || profile.VehicleType == nil {
		return ""
	}

	r.mu.Lock()
	r.riderVehicle[riderID] = *profile.VehicleType
	r.mu.Unlock()
	return *profile.VehicleType
}

func (r *appRidersManager) Unregister(riderID uuid.UUID, conn ridersPorts.RiderConn) {
	r.mu.Lock()
	if stored, ok := r.availableRiders[riderID]; ok && stored == conn {
		if loc, hasLoc := r.riderLocation[riderID]; hasLoc {
			r.unindexCellLocked(riderID, loc.H3Cell)
		}
		delete(r.availableRiders, riderID)
		delete(r.riderMode, riderID)
		delete(r.riderLocation, riderID)
		delete(r.riderVehicle, riderID)
	}
	r.mu.Unlock()

	conn.Close()
}

func (r *appRidersManager) UpdateLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64) error {
	// LatLngToCell acepta cualquier coordenada del planeta: el área de servicio se exige aparte.
	if !geo.InVenezuela(lat, lng) {
		return errcustom.ErrInvalidCoordinates
	}

	cell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	if err != nil {
		return errcustom.ErrInvalidCoordinates
	}

	r.mu.Lock()
	prev, hadLocation := r.riderLocation[riderID]
	r.riderLocation[riderID] = riderModels.RiderLocation{
		Latitude:  lat,
		Longitude: lng,
		H3Cell:    cell,
		UpdatedAt: time.Now(),
	}
	prevCell := ""
	if hadLocation {
		prevCell = prev.H3Cell
	}
	inTripsMode := r.riderMode[riderID] == riderModels.RiderModeTrips
	r.reindexRiderLocked(riderID, prevCell, cell, inTripsMode)
	assignment, onTrip := r.riderTrips[riderID]
	r.mu.Unlock()

	if onTrip && r.customers != nil {
		r.customers.NotifyRiderLocation(ctx, assignment.customerID, assignment.tripID, lat, lng)
	}
	r.notifyOrderLocations(ctx, riderID, lat, lng, time.Now())
	if inTripsMode && (!hadLocation || prev.H3Cell != cell) {
		r.offerOpenTripsToRider(ctx, riderID, cell)
	}
	return nil
}

// SetOrderCustomerNotifier enlaza el manager WS de orders, que no se puede inyectar
// en New sin crear un ciclo de DI.
func (r *appRidersManager) SetOrderCustomerNotifier(n ridersPorts.OrderCustomerNotifier) {
	r.mu.Lock()
	r.orderNotifier = n
	r.mu.Unlock()
}

// notifyOrderLocations envía la ubicación del rider al cliente de cada orden que
// tiene asignada (un mensaje por orden). Solo hay órdenes asignadas una vez que
// el rider las aceptó, así que nunca se emite antes de eso.
func (r *appRidersManager) notifyOrderLocations(ctx context.Context, riderID uuid.UUID, lat, lng float64, at time.Time) {
	r.mu.RLock()
	notifier := r.orderNotifier
	targets := make(map[uuid.UUID]uuid.UUID, len(r.assignedOrders[riderID]))
	for _, orderID := range r.assignedOrders[riderID] {
		if customerID, ok := r.orderCustomer[orderID]; ok {
			targets[orderID] = customerID
		}
	}
	r.mu.RUnlock()

	if notifier == nil {
		return
	}
	for orderID, customerID := range targets {
		notifier.NotifyRiderOrderLocation(ctx, customerID, orderID, lat, lng, at)
	}
}

// forgetOrderLocked saca una orden terminada de assignedOrders y orderCustomer.
// Debe llamarse con r.mu tomado para escritura.
func (r *appRidersManager) forgetOrderLocked(orderID uuid.UUID) {
	delete(r.orderCustomer, orderID)
	for riderID, ids := range r.assignedOrders {
		if !slices.Contains(ids, orderID) {
			continue
		}
		ids = slices.DeleteFunc(slices.Clone(ids), func(id uuid.UUID) bool { return id == orderID })
		if len(ids) == 0 {
			delete(r.assignedOrders, riderID)
		} else {
			r.assignedOrders[riderID] = ids
		}
	}
}

func isTerminalOrderStatus(status string) bool {
	switch ordersModels.OrderStatus(status) {
	case ordersModels.StatusCompleted, ordersModels.StatusCancelled, ordersModels.StatusRejected:
		return true
	default:
		return false
	}
}

// offerOpenTripsToRider le reenvía a un rider que llega tarde (primera ubicación
// de la conexión) o que cambia de celda los viajes que siguen buscando conductor
// dentro del radio de su ronda actual. Excluye los que canceló y los que ya se
// ofertaron en una ronda anterior sin que él estuviera en ella; a quien ya está
// en la ronda vigente se le reenvía la oferta (por si perdió la conexión).
func (r *appRidersManager) offerOpenTripsToRider(ctx context.Context, riderID uuid.UUID, cell string) {
	r.mu.RLock()
	conn, connected := r.availableRiders[riderID]
	// La bici no lleva pasajeros: tampoco por este camino, que corre en cada cambio de celda.
	if !connected || riderModels.IsBike(r.riderVehicle[riderID]) || len(r.tripOfferTrip) == 0 || r.riderBusyLocked(riderID) {
		r.mu.RUnlock()
		return
	}
	open := make([]tripsModels.Trip, 0, len(r.tripOfferTrip))
	for tripID, trip := range r.tripOfferTrip {
		if trip.PickupH3Cell == nil {
			continue
		}
		if _, excluded := r.tripExcluded[tripID][riderID]; excluded {
			continue
		}
		_, offeredNow := r.tripOffers[tripID][riderID]
		if _, offeredBefore := r.tripAlreadyOffered[tripID][riderID]; offeredBefore && !offeredNow {
			continue
		}
		maxRings := tripSearchMaxRings + r.tripOfferRounds[tripID] - 1
		if !cellWithinRings(*trip.PickupH3Cell, cell, maxRings) {
			continue
		}
		open = append(open, trip)
	}
	r.mu.RUnlock()

	for _, trip := range open {
		current, err := r.tripsRepo.GetTripByID(ctx, trip.ID)
		if err != nil {
			slog.Error("error reading trip to offer to late rider", "tripID", trip.ID, "riderID", riderID, "error", err)
			continue
		}
		if current.Status != tripsModels.TripStatusSearching {
			continue
		}
		data, err := json.Marshal(tripBroadcast{Type: "trip_offer", Trip: r.tripOfferWithPassenger(ctx, &trip)})
		if err != nil {
			slog.Error("error marshaling trip_offer for late rider", "error", err)
			continue
		}

		r.mu.Lock()
		if _, live := r.tripOfferTrip[trip.ID]; !live {
			r.mu.Unlock()
			continue
		}
		if r.tripOffers[trip.ID] == nil {
			r.tripOffers[trip.ID] = make(map[uuid.UUID]struct{})
		}
		r.tripOffers[trip.ID][riderID] = struct{}{}
		if r.tripAlreadyOffered[trip.ID] == nil {
			r.tripAlreadyOffered[trip.ID] = make(map[uuid.UUID]struct{})
		}
		r.tripAlreadyOffered[trip.ID][riderID] = struct{}{}
		r.mu.Unlock()

		conn.Send(data)
	}
}

func cellWithinRings(originCell, cell string, rings int) bool {
	cells, err := geo.GridDisk(originCell, rings)
	if err != nil {
		return false
	}
	return slices.Contains(cells, cell)
}

// SendActiveTrips reenvía al rider que se reconecta en modo trips los viajes que
// ya tiene asignados y vuelve a enlazar su ubicación con el cliente (riderTrips
// vive en memoria y se pierde al reconectar).
func (r *appRidersManager) SendActiveTrips(ctx context.Context, riderID uuid.UUID, conn ridersPorts.RiderConn) error {
	trips, err := r.tripsRepo.ListActiveTripsByRider(ctx, riderID)
	if err != nil {
		return err
	}
	if len(trips) == 0 {
		return nil
	}

	for i := range trips {
		data, err := json.Marshal(tripBroadcast{Type: "trip_status", Trip: r.riderTripDTO(ctx, &trips[i])})
		if err != nil {
			slog.Error("error marshaling active trip for rider", "tripID", trips[i].ID, "error", err)
			continue
		}
		conn.Send(data)
	}
	r.rememberRiderTrip(riderID, trips[0].ID, trips[0].CustomerID)
	return nil
}

// ActiveTrips devuelve por REST los viajes en curso del rider, sin códigos.
func (r *appRidersManager) ActiveTrips(ctx context.Context, riderID uuid.UUID) ([]tripsDtos.TripResponse, error) {
	trips, err := r.tripsRepo.ListActiveTripsByRider(ctx, riderID)
	if err != nil {
		return nil, err
	}
	if len(trips) == 0 {
		return []tripsDtos.TripResponse{}, nil
	}

	out := make([]tripsDtos.TripResponse, len(trips))
	for i := range trips {
		out[i] = r.riderTripDTO(ctx, &trips[i])
	}
	return out, nil
}

func (r *appRidersManager) SnapshotTripsRiders() []riderModels.RiderLocationEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	out := make([]riderModels.RiderLocationEntry, 0, len(r.riderLocation))
	for riderID, loc := range r.riderLocation {
		if r.riderMode[riderID] != riderModels.RiderModeTrips {
			continue
		}
		if now.Sub(loc.UpdatedAt) > riderModels.TripMatchRiderMaxAge {
			continue
		}
		// La bici no lleva pasajeros. Un vehículo sin cargar no excluye: hay riders legacy sin el campo.
		if riderModels.IsBike(r.riderVehicle[riderID]) {
			continue
		}
		out = append(out, riderModels.RiderLocationEntry{RiderID: riderID, Location: loc})
	}
	return out
}

func (r *appRidersManager) BroadcastNewOrder(ctx context.Context, order ordersModels.Order) {
	msg := r.orderBroadcast(ctx, "new_order", order, false)
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("error marshaling new_order broadcast", "error", err)
		return
	}

	// El shop ya viene de orderBroadcast: el filtro por vehículo no cuesta una consulta extra.
	shopType := ""
	if msg.Shop != nil {
		shopType = msg.Shop.Type
	}

	r.mu.RLock()
	clients := make([]ridersPorts.RiderConn, 0, len(r.availableRiders))
	for riderID, c := range r.availableRiders {
		if r.riderMode[riderID] == riderModels.RiderModeTrips {
			continue
		}
		if !riderModels.VehicleCanCarryShopType(r.riderVehicle[riderID], shopType) {
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
	data, err := json.Marshal(r.orderBroadcast(ctx, "order_update", order, true))
	if err != nil {
		slog.Error("error marshaling order_update for rider", "error", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for riderID, orderIDs := range r.assignedOrders {
		if slices.Contains(orderIDs, order.ID) {
			if c, ok := r.availableRiders[riderID]; ok {
				c.Send(data)
			}
		}
	}
	if isTerminalOrderStatus(order.Status) {
		r.forgetOrderLocked(order.ID)
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

	if err := r.ensureVehicleCanCarry(ctx, riderID, *order); err != nil {
		return err
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
	r.orderCustomer[orderID] = order.CustomerID
	loc, hasLoc := r.riderLocation[riderID]
	notifier := r.orderNotifier
	r.mu.Unlock()

	deliveryfee.ApplyBsToOrder(ctx, r.bcv, order)
	accepted, _ := json.Marshal(r.orderBroadcast(ctx, "order_accepted", *order, true))
	conn.Send(accepted)

	// El pin del rider aparece en el cliente apenas acepta, sin esperar al siguiente tick de ubicación.
	if hasLoc && notifier != nil {
		notifier.NotifyRiderOrderLocation(ctx, order.CustomerID, orderID, loc.Latitude, loc.Longitude, loc.UpdatedAt)
	}

	return nil
}

// ensureVehicleCanCarry corta el pedido que no entra en el vehículo del rider.
// Es el único de los tres caminos que es regla: filtrar los broadcasts es UX.
func (r *appRidersManager) ensureVehicleCanCarry(ctx context.Context, riderID uuid.UUID, order ordersModels.Order) error {
	// Un shop que no se pudo leer no bloquea: mismo criterio fail-open que el vehículo sin cargar.
	shop := r.shopForOrder(ctx, order)
	if shop == nil {
		return nil
	}

	r.mu.RLock()
	vehicle := r.riderVehicle[riderID]
	r.mu.RUnlock()

	if riderModels.VehicleCanCarryShopType(vehicle, shop.Type) {
		return nil
	}
	return errcustom.ErrVehicleCannotCarryOrder
}

// ensureVehicleCanCarryTrip corta el viaje de pasajeros que la bici no puede llevar.
func (r *appRidersManager) ensureVehicleCanCarryTrip(riderID uuid.UUID) error {
	r.mu.RLock()
	vehicle := r.riderVehicle[riderID]
	r.mu.RUnlock()

	// Un vehículo sin cargar no bloquea: mismo criterio fail-open que los pedidos.
	if !riderModels.IsBike(vehicle) {
		return nil
	}
	return errcustom.ErrVehicleCannotCarryTrip
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
	// ShopToCustomerKm es la distancia en línea recta tienda → destino de entrega.
	ShopToCustomerKm *float64 `json:"shop_to_customer_km,omitempty"`
}

// orderBroadcast arma el evento de orden para riders. assigned sigue la misma
// semántica que en NewRiderOrder.
func (r *appRidersManager) orderBroadcast(ctx context.Context, eventType string, order ordersModels.Order, assigned bool) riderBroadcast {
	shop := r.shopForOrder(ctx, order)
	msg := riderBroadcast{
		Type:     eventType,
		Order:    riderModels.NewRiderOrder(order, assigned),
		Shop:     shop,
		Customer: customerLocation(order, assigned),
	}
	if shop != nil {
		km := geo.HaversineKm(shop.Latitude, shop.Longitude, order.Latitude, order.Longitude)
		msg.ShopToCustomerKm = &km
	}
	return msg
}

func (r *appRidersManager) OfferTrip(_ context.Context, trip tripsModels.Trip) {
	r.offerTripRound(trip, 1)
}

func (r *appRidersManager) offerTripRound(trip tripsModels.Trip, round int) {
	maxRings := tripSearchMaxRings + round - 1
	candidates := r.nearestTripsRiders(trip, r.offerExclusions(trip.ID), maxRings)
	if len(candidates) == 0 {
		// Sin celda H3 el viaje nunca va a matchear: no tiene sentido esperar.
		if trip.PickupH3Cell == nil {
			r.expireTripIfStillSearching(trip.ID)
			return
		}
		// Ronda vacía: igual espera tripEmptyRoundTimeout (puede conectarse un rider
		// elegible) y cuenta como ronda; los ofertados de la ronda anterior ya
		// no pueden aceptar.
		r.mu.Lock()
		if _, withdrawn := r.tripWithdrawn[trip.ID]; withdrawn {
			r.mu.Unlock()
			return
		}
		r.tripOffers[trip.ID] = make(map[uuid.UUID]struct{})
		r.tripOfferRounds[trip.ID] = round
		r.tripOfferTrip[trip.ID] = trip
		r.mu.Unlock()
		r.scheduleOfferTimeout(trip, tripEmptyRoundTimeout)
		return
	}

	data, err := json.Marshal(tripBroadcast{Type: "trip_offer", Trip: r.tripOfferWithPassenger(context.Background(), &trip)})
	if err != nil {
		slog.Error("error marshaling trip_offer broadcast", "error", err)
		return
	}

	r.mu.Lock()
	if _, withdrawn := r.tripWithdrawn[trip.ID]; withdrawn {
		r.mu.Unlock()
		return
	}
	offered := make(map[uuid.UUID]struct{}, len(candidates))
	if r.tripAlreadyOffered[trip.ID] == nil {
		r.tripAlreadyOffered[trip.ID] = make(map[uuid.UUID]struct{})
	}
	for _, c := range candidates {
		offered[c.RiderID] = struct{}{}
		r.tripAlreadyOffered[trip.ID][c.RiderID] = struct{}{}
	}
	r.tripOffers[trip.ID] = offered
	r.tripOfferRounds[trip.ID] = round
	r.tripOfferTrip[trip.ID] = trip
	r.mu.Unlock()

	r.mu.RLock()
	if _, withdrawn := r.tripWithdrawn[trip.ID]; withdrawn {
		r.mu.RUnlock()
		return
	}
	for _, c := range candidates {
		if conn, ok := r.availableRiders[c.RiderID]; ok {
			conn.Send(data)
		}
	}
	r.mu.RUnlock()

	r.scheduleOfferTimeout(trip, tripAcceptTimeout)
}

func (r *appRidersManager) scheduleOfferTimeout(trip tripsModels.Trip, timeout time.Duration) {
	timer := time.AfterFunc(timeout, func() {
		r.onTripOfferTimeout(trip)
	})
	r.mu.Lock()
	if _, withdrawn := r.tripWithdrawn[trip.ID]; withdrawn {
		timer.Stop()
		r.mu.Unlock()
		return
	}
	if prev, ok := r.tripTimers[trip.ID]; ok {
		prev.Stop()
	}
	r.tripTimers[trip.ID] = timer
	r.mu.Unlock()
}

func (r *appRidersManager) onTripOfferTimeout(offered tripsModels.Trip) {
	ctx := context.Background()
	current, err := r.tripsRepo.GetTripByID(ctx, offered.ID)
	if err != nil {
		slog.Error("error reading trip on offer timeout", "tripID", offered.ID, "error", err)
		return
	}
	if current.Status != tripsModels.TripStatusSearching {
		r.clearTripOfferState(current.ID)
		return
	}
	if current.PickupH3Cell == nil {
		current.PickupH3Cell = offered.PickupH3Cell
		current.PickupLatitude = offered.PickupLatitude
		current.PickupLongitude = offered.PickupLongitude
	}
	if r.offerRound(current.ID) >= tripOfferMaxRounds {
		r.expireTripIfStillSearching(current.ID)
		return
	}
	r.offerTripRound(*current, r.offerRound(current.ID)+1)
}

func (r *appRidersManager) offerRound(tripID uuid.UUID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tripOfferRounds[tripID]
}

func (r *appRidersManager) offerExclusions(tripID uuid.UUID) map[uuid.UUID]struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[uuid.UUID]struct{})
	for id := range r.tripExcluded[tripID] {
		out[id] = struct{}{}
	}
	for id := range r.tripAlreadyOffered[tripID] {
		out[id] = struct{}{}
	}
	for id := range r.riderTrips {
		out[id] = struct{}{}
	}
	for id := range r.acceptingRiders {
		out[id] = struct{}{}
	}
	return out
}

// riderBusyLocked requiere r.mu tomado.
func (r *appRidersManager) riderBusyLocked(riderID uuid.UUID) bool {
	if _, onTrip := r.riderTrips[riderID]; onTrip {
		return true
	}
	_, accepting := r.acceptingRiders[riderID]
	return accepting
}

func (r *appRidersManager) beginAccepting(riderID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.riderBusyLocked(riderID) {
		return false
	}
	r.acceptingRiders[riderID] = struct{}{}
	return true
}

func (r *appRidersManager) endAccepting(riderID uuid.UUID) {
	r.mu.Lock()
	delete(r.acceptingRiders, riderID)
	r.mu.Unlock()
}

func (r *appRidersManager) expireTripIfStillSearching(tripID uuid.UUID) {
	ctx := context.Background()
	ok, err := r.tripsRepo.UpdateTripStatus(ctx, tripID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver)
	if err != nil {
		slog.Error("error expiring trip to no_driver", "tripID", tripID, "error", err)
	}

	r.clearTripOfferState(tripID)
	if err != nil || !ok {
		return
	}
	// El motivo se escribe sin cargar el viaje: aunque no haya nada que
	// reembolsar ni a quién avisar, tiene que quedar registrado.
	if reasonErr := r.tripsRepo.SaveTripCancelReason(ctx, tripID, tripsModels.CancelReasonNoDriver); reasonErr != nil {
		slog.Error("save trip cancel reason", "tripID", tripID, "reason", tripsModels.CancelReasonNoDriver, "error", reasonErr)
	}
	if r.customers == nil && r.balance == nil {
		return
	}
	trip, getErr := r.tripsRepo.GetTripByID(ctx, tripID)
	if getErr != nil {
		slog.Error("error loading expired trip", "tripID", tripID, "error", getErr)
		return
	}
	if trip.BalancePaidBs != nil && *trip.BalancePaidBs > 0 && r.balance != nil {
		if refundErr := r.balance.RefundBalancePayment(ctx, trip.CustomerID, trip.ID, tripsModels.ReferenceTypeTrip, *trip.BalancePaidBs); refundErr != nil {
			slog.Error("refund balance on no_driver", "tripID", trip.ID, "error", refundErr)
		}
	}
	tripsApp.CreditUnservedPagoMovil(ctx, r.balance, trip, r.tripsRepo.SaveTripPayment)
	trip.SetCancelReason(tripsModels.CancelReasonNoDriver)
	r.notifyCustomer(ctx, trip)
	r.publishTripNoDriver(ctx, trip)
}

func (r *appRidersManager) publishTripNoDriver(ctx context.Context, trip *tripsModels.Trip) {
	if r.stream == nil {
		return
	}
	event := tripsModels.DataEventTrip{}
	event.BuildFromTrip(tripsModels.EventTripNoDriver, *trip)
	if err := r.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing trip_no_driver event", "tripID", trip.ID, "error", err)
	}
}

func (r *appRidersManager) clearTripOfferState(tripID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tripOffers, tripID)
	delete(r.tripAlreadyOffered, tripID)
	delete(r.tripOfferRounds, tripID)
	delete(r.tripOfferTrip, tripID)
	delete(r.tripExcluded, tripID)
	if timer, ok := r.tripTimers[tripID]; ok {
		timer.Stop()
		delete(r.tripTimers, tripID)
	}
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

func (r *appRidersManager) nearestTripsRiders(trip tripsModels.Trip, exclude map[uuid.UUID]struct{}, maxRings int) []riderModels.RiderLocationEntry {
	if trip.PickupH3Cell == nil {
		return nil
	}

	byCell := make(map[string][]riderModels.RiderLocationEntry)
	for _, e := range r.SnapshotTripsRiders() {
		if _, skip := exclude[e.RiderID]; skip {
			continue
		}
		byCell[e.Location.H3Cell] = append(byCell[e.Location.H3Cell], e)
	}

	for k := 0; k <= maxRings; k++ {
		cells, err := geo.GridDisk(*trip.PickupH3Cell, k)
		if err != nil {
			slog.Error("error expanding H3 rings for trip matching", "tripID", trip.ID, "error", err)
			return nil
		}

		matched := make([]riderModels.RiderLocationEntry, 0, tripOfferCandidateCount)
		for _, cell := range cells {
			matched = append(matched, byCell[cell]...)
		}

		if len(matched) < tripOfferCandidateCount && k < maxRings {
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
	// Filtrar las ofertas es UX; esto es la regla, como ensureVehicleCanCarry en los pedidos.
	if err := r.ensureVehicleCanCarryTrip(riderID); err != nil {
		return err
	}
	if !r.beginAccepting(riderID) {
		return errcustom.ErrRiderBusy
	}
	defer r.endAccepting(riderID)

	pickupCode, err := generateTripCode()
	if err != nil {
		return errcustom.ErrUnexpectedError
	}
	ok, err := r.tripsRepo.AssignTripRider(ctx, tripID, riderID, pickupCode)
	if err != nil {
		return err
	}
	if !ok {
		return r.rejectAssignment(ctx, tripID, conn)
	}

	r.clearTripOfferState(tripID)

	acceptedAt := time.Now()
	trip.RiderID = &riderID
	trip.Status = tripsModels.TripStatusAccepted
	trip.AcceptedAt = &acceptedAt
	trip.PickupCode = &pickupCode
	r.rememberRiderTrip(riderID, trip.ID, trip.CustomerID)
	accepted, _ := json.Marshal(tripBroadcast{Type: "trip_accepted", Trip: r.riderTripDTO(ctx, trip)})
	conn.Send(accepted)
	r.notifyCustomer(ctx, trip)

	return nil
}

// rejectAssignment distingue si otro rider ganó el viaje o si la BD rechazó al
// rider por tener ya un viaje en curso (el viaje sigue en searching).
func (r *appRidersManager) rejectAssignment(ctx context.Context, tripID uuid.UUID, conn ridersPorts.RiderConn) error {
	current, err := r.tripsRepo.GetTripByID(ctx, tripID)
	if err == nil && current.Status == tripsModels.TripStatusSearching {
		return errcustom.ErrRiderBusy
	}
	msg, _ := json.Marshal(map[string]string{"type": "trip_taken", "trip_id": tripID.String()})
	conn.Send(msg)
	return nil
}

// CancelAcceptedTrip libera el viaje antes de recoger al pasajero y lo vuelve
// a ofertar sin el rider que canceló. in_progress queda fuera: sería un
// incidente, no una re-oferta.
func (r *appRidersManager) CancelAcceptedTrip(ctx context.Context, riderID, tripID uuid.UUID, conn ridersPorts.RiderConn) error {
	trip, err := r.tripsRepo.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.RiderID == nil || *trip.RiderID != riderID {
		return errcustom.ErrInvalidTransition
	}
	if trip.Status != tripsModels.TripStatusAccepted && trip.Status != tripsModels.TripStatusArriving {
		return errcustom.ErrInvalidTransition
	}

	ok, err := r.tripsRepo.ReleaseTripRider(ctx, tripID, riderID)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidTransition
	}

	r.excludeRider(tripID, riderID)
	slog.Info("rider cancelled accepted trip", "tripID", tripID, "riderID", riderID, "fromStatus", string(trip.Status), "at", time.Now().UTC())

	r.forgetRiderTrip(riderID)
	trip.Status = tripsModels.TripStatusSearching
	trip.RiderID = nil
	trip.AcceptedAt = nil
	trip.PickupCode = nil
	released, _ := json.Marshal(tripBroadcast{Type: "trip_status", Trip: r.riderTripDTO(ctx, trip)})
	conn.Send(released)
	r.notifyCustomer(ctx, trip)

	r.OfferTrip(ctx, *trip)
	return nil
}

func (r *appRidersManager) excludeRider(tripID, riderID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.tripExcluded[tripID]
	if !ok {
		set = make(map[uuid.UUID]struct{})
		r.tripExcluded[tripID] = set
	}
	set[riderID] = struct{}{}
}

// AdvanceTrip mueve el trip asignado al siguiente estado. Solo el rider
// asignado puede avanzarlo; in_progress→completed se resuelve en
// completeTrip porque además acredita la ganancia del rider.
// Vive en riders/app (no en trips/app) a propósito, igual que AcceptTrip:
// ver "Trips & Riders" en CLAUDE.md para el porqué de este split.
func (r *appRidersManager) AdvanceTrip(ctx context.Context, riderID, tripID uuid.UUID, code string, conn ridersPorts.RiderConn) error {
	trip, err := r.tripsRepo.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.RiderID == nil || *trip.RiderID != riderID {
		return errcustom.ErrInvalidTransition
	}

	if trip.Status == tripsModels.TripStatusInProgress {
		return r.completeTrip(ctx, trip, strings.TrimSpace(code), conn)
	}
	if trip.Status == tripsModels.TripStatusArriving {
		return r.startTrip(ctx, trip, riderID, strings.TrimSpace(code), conn)
	}

	next, ok := tripForwardStatus[trip.Status]
	if !ok {
		return errcustom.ErrInvalidTransition
	}

	updated, err := r.tripsRepo.UpdateTripStatus(ctx, tripID, trip.Status, next)
	if err != nil {
		return err
	}
	if !updated {
		return errcustom.ErrInvalidTransition
	}

	trip.Status = next
	data, _ := json.Marshal(tripBroadcast{Type: "trip_status", Trip: r.riderTripDTO(ctx, trip)})
	conn.Send(data)
	r.notifyCustomer(ctx, trip)

	return nil
}

func (r *appRidersManager) startTrip(ctx context.Context, trip *tripsModels.Trip, riderID uuid.UUID, pickupCode string, conn ridersPorts.RiderConn) error {
	if pickupCode == "" {
		return errcustom.ErrInvalidCode
	}
	dropoffCode, err := generateTripCode()
	if err != nil {
		return errcustom.ErrUnexpectedError
	}
	updated, err := r.tripsRepo.StartTripWithPickupCode(ctx, trip.ID, riderID, pickupCode, dropoffCode)
	if err != nil {
		return err
	}
	if !updated {
		return errcustom.ErrInvalidCode
	}

	usedAt := time.Now()
	trip.Status = tripsModels.TripStatusInProgress
	trip.PickupCodeUsedAt = &usedAt
	trip.StartedAt = &usedAt
	trip.DropoffCode = &dropoffCode
	data, _ := json.Marshal(tripBroadcast{Type: "trip_status", Trip: r.riderTripDTO(ctx, trip)})
	conn.Send(data)
	r.notifyCustomer(ctx, trip)
	return nil
}

// completeTrip usa el fare que ya quedó fijado en trip.Fare desde RequestTrip, no lo recalcula.
func (r *appRidersManager) completeTrip(ctx context.Context, trip *tripsModels.Trip, dropoffCode string, conn ridersPorts.RiderConn) error {
	if !r.beginCompleting(trip.ID) {
		return errcustom.ErrInvalidTransition
	}
	defer r.endCompleting(trip.ID)

	if trip.Fare == nil {
		return errcustom.ErrUnexpectedError
	}
	if dropoffCode == "" {
		return errcustom.ErrInvalidCode
	}
	fare := *trip.Fare

	updated, err := r.tripsRepo.CompleteTrip(ctx, trip.ID, fare, dropoffCode)
	if err != nil {
		return err
	}
	if !updated {
		return errcustom.ErrInvalidCode
	}

	completedAt := time.Now()
	trip.Status = tripsModels.TripStatusCompleted
	trip.CompletedAt = &completedAt
	trip.DropoffCodeUsedAt = &completedAt
	trip.ApplyFrozenBsFare()
	data, _ := json.Marshal(tripBroadcast{Type: "trip_status", Trip: r.riderTripDTO(ctx, trip)})
	conn.Send(data)
	if trip.RiderID != nil {
		r.forgetRiderTrip(*trip.RiderID)
	}
	r.notifyCustomer(ctx, trip)

	// El pago del rider sale de la tarifa completa; con descuento, el efectivo puede no cubrirlo y la plataforma pone la diferencia.
	var earnErr error
	gross := trip.GrossFareUSD()
	cashUSD := tripCashCollectedUSD(trip, fare)
	partialCash := cashUSD > 0 && gross-cashUSD > 0.01
	if partialCash {
		earnErr = r.tripEarnings.RecordTripCashCollected(ctx, *trip.RiderID, trip.ID, gross, cashUSD, r.bcvRate(ctx))
	}
	if cashUSD > 0 && !partialCash {
		earnErr = r.tripEarnings.RecordTripCash(ctx, *trip.RiderID, trip.ID, gross, r.bcvRate(ctx))
	}
	if cashUSD <= 0 {
		earnErr = r.tripEarnings.RecordTripEarning(ctx, *trip.RiderID, trip.ID, gross, r.bcvRate(ctx))
	}
	if earnErr != nil {
		slog.Error("error recording trip earning", "tripID", trip.ID, "riderID", *trip.RiderID, "error", earnErr)
	}

	return nil
}

func (r *appRidersManager) rememberRiderTrip(riderID, tripID, customerID uuid.UUID) {
	r.mu.Lock()
	r.riderTrips[riderID] = riderAssignment{tripID: tripID, customerID: customerID}
	r.mu.Unlock()
}

func (r *appRidersManager) forgetRiderTrip(riderID uuid.UUID) {
	r.mu.Lock()
	delete(r.riderTrips, riderID)
	r.mu.Unlock()
}

// NotifyTripCancelled avisa al rider asignado y a quienes tienen la oferta,
// deja de reenviar la ubicación del asignado y retira la búsqueda en memoria.
// El mensaje no incluye motivo ni quién canceló.
func (r *appRidersManager) NotifyTripCancelled(_ context.Context, trip tripsModels.Trip) {
	payload, err := json.Marshal(tripCancelledNotice{
		Type:   "trip_cancelled",
		TripID: trip.ID.String(),
	})
	if err != nil {
		slog.Error("error marshaling trip_cancelled", "tripID", trip.ID, "error", err)
		return
	}

	r.mu.Lock()
	recipients := make(map[uuid.UUID]ridersPorts.RiderConn)
	collect := func(riderID uuid.UUID) {
		if conn, ok := r.availableRiders[riderID]; ok {
			recipients[riderID] = conn
		}
	}
	for riderID := range r.tripOffers[trip.ID] {
		collect(riderID)
	}
	for riderID := range r.tripAlreadyOffered[trip.ID] {
		collect(riderID)
	}
	if trip.RiderID != nil {
		collect(*trip.RiderID)
		delete(r.riderTrips, *trip.RiderID)
	}
	r.tripWithdrawn[trip.ID] = struct{}{}
	delete(r.tripOffers, trip.ID)
	delete(r.tripAlreadyOffered, trip.ID)
	delete(r.tripOfferRounds, trip.ID)
	delete(r.tripOfferTrip, trip.ID)
	delete(r.tripExcluded, trip.ID)
	if timer, ok := r.tripTimers[trip.ID]; ok {
		timer.Stop()
		delete(r.tripTimers, trip.ID)
	}
	r.mu.Unlock()

	for _, conn := range recipients {
		conn.Send(payload)
	}
}

func (r *appRidersManager) notifyCustomer(ctx context.Context, trip *tripsModels.Trip) {
	if r.customers == nil || trip == nil {
		return
	}
	r.customers.NotifyTrip(ctx, *trip)
}

func tripCashCollectedUSD(trip *tripsModels.Trip, fare float64) float64 {
	if trip == nil {
		return 0
	}
	if trip.CashAmountBs != nil && *trip.CashAmountBs > 0 && trip.BcvRate != nil && *trip.BcvRate > 0 {
		return *trip.CashAmountBs / *trip.BcvRate
	}
	if trip.MethodPayment == tripsModels.MethodEfectivo {
		return fare
	}
	return 0
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

type tripCancelledNotice struct {
	Type   string `json:"type"`
	TripID string `json:"trip_id"`
}

// AdvanceTripOfferTimeout corre el mismo camino que el timer de oferta (tripAcceptTimeout o tripEmptyRoundTimeout).
// Los tests lo usan para pasar de ronda sin esperar el timeout real.
func AdvanceTripOfferTimeout(svc ridersPorts.RidersApp, trip tripsModels.Trip) {
	mgr := svc.(*appRidersManager)
	mgr.mu.Lock()
	if timer, ok := mgr.tripTimers[trip.ID]; ok {
		timer.Stop()
		delete(mgr.tripTimers, trip.ID)
	}
	mgr.mu.Unlock()
	mgr.onTripOfferTimeout(trip)
}

// TripSearchMaxRingsForTest expone el tope de anillos de la primera ronda, para
// que los tests ubiquen riders dentro o fuera del radio sin fijar el número.
const TripSearchMaxRingsForTest = tripSearchMaxRings

// StopTripOfferTimerForTest detiene el timer de oferta para que no dispare
// después de que el test ya terminó.
func StopTripOfferTimerForTest(svc ridersPorts.RidersApp, tripID uuid.UUID) {
	mgr := svc.(*appRidersManager)
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if timer, ok := mgr.tripTimers[tripID]; ok {
		timer.Stop()
		delete(mgr.tripTimers, tripID)
	}
}
