package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
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
	"Encargalo.app-api.go/internal/shared/tripfare"
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
	// offlineOrderFallbackRadiusKm es el radio del respaldo de última ubicación
	// conocida para pedidos: a diferencia de viajes, new_order no filtra por
	// distancia a los riders conectados, así que no hay un radio "en vivo" que
	// reusar — se fija uno razonable para el reparto a pie/moto/bici.
	offlineOrderFallbackRadiusKm = 5.0
	// lastKnownLocationTimeout acota cuánto puede vivir cada goroutine que
	// toca riders.last_known_locations (guardado en cada update_location,
	// búsqueda en cada ronda de oferta) — sin tope, una degradación de
	// Postgres las acumula indefinidamente.
	lastKnownLocationTimeout = 5 * time.Second
)

type appRidersManager struct {
	mu                 sync.RWMutex
	availableRiders    map[uuid.UUID]ridersPorts.RiderConn
	assignedOrders     map[uuid.UUID][]uuid.UUID // riderID → []orderID
	orderCustomer      map[uuid.UUID]uuid.UUID   // orderID → customerID, para avisarle la ubicación del rider
	riderMode          map[uuid.UUID]riderModels.RiderMode
	riderLocation      map[uuid.UUID]riderModels.RiderLocation
	riderVehicle       map[uuid.UUID]string
	tripOffers         map[uuid.UUID]map[uuid.UUID]struct{}
	tripAlreadyOffered map[uuid.UUID]map[uuid.UUID]struct{}
	tripOfferRounds    map[uuid.UUID]int
	tripOfferTrip      map[uuid.UUID]tripsModels.Trip
	tripExcluded       map[uuid.UUID]map[uuid.UUID]struct{}
	// tripWithdrawn marca viajes que el cliente ya canceló. Una ronda que ya
	// pasó el chequeo de status no puede volver a publicar la oferta.
	tripWithdrawn   map[uuid.UUID]struct{}
	completingTrips map[uuid.UUID]struct{}
	// acceptingRiders evita que un rider acepte dos viajes a la vez antes de quedar en riderTrips.
	acceptingRiders map[uuid.UUID]struct{}
	// pausedRiders espeja en memoria la pausa de nuevas asignaciones (fuente de
	// verdad en Redis, ver RiderPresenceStore) para no pedirle a Redis en cada oferta.
	pausedRiders  map[uuid.UUID]struct{}
	repo          ridersPorts.RidersRepo
	stream        ridersPorts.RidersStream
	bcv           bcvPorts.BcvApp
	tripsRepo     ridersPorts.TripsStore
	tripEarnings  ridersPorts.TripEarningRecorder
	customers     tripsPorts.CustomerTripNotifier
	balance       paymentsPorts.CustomerBalanceCreditor
	passengers    ridersPorts.PassengerLookup
	vehicles      ridersPorts.VehicleLookup
	presence      ridersPorts.RiderPresenceStore
	riderTrips    map[uuid.UUID]riderAssignment
	orderNotifier ridersPorts.OrderCustomerNotifier
	offerTimeouts ridersPorts.TripOfferTimeoutStore
	lastKnown     ridersPorts.LastKnownLocationStore
	push          ridersPorts.PushNotifier
}

type riderAssignment struct {
	tripID     uuid.UUID
	customerID uuid.UUID
}

func New(repo ridersPorts.RidersRepo, stream ridersPorts.RidersStream, bcv bcvPorts.BcvApp, tripsRepo ridersPorts.TripsStore, tripEarnings ridersPorts.TripEarningRecorder, customers tripsPorts.CustomerTripNotifier, balance paymentsPorts.CustomerBalanceCreditor, passengers ridersPorts.PassengerLookup, vehicles ridersPorts.VehicleLookup, presence ridersPorts.RiderPresenceStore, offerTimeouts ridersPorts.TripOfferTimeoutStore, lastKnown ridersPorts.LastKnownLocationStore, push ridersPorts.PushNotifier) ridersPorts.RidersApp {
	return &appRidersManager{
		availableRiders:    make(map[uuid.UUID]ridersPorts.RiderConn),
		assignedOrders:     make(map[uuid.UUID][]uuid.UUID),
		orderCustomer:      make(map[uuid.UUID]uuid.UUID),
		riderMode:          make(map[uuid.UUID]riderModels.RiderMode),
		riderLocation:      make(map[uuid.UUID]riderModels.RiderLocation),
		riderVehicle:       make(map[uuid.UUID]string),
		tripOffers:         make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripAlreadyOffered: make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripOfferRounds:    make(map[uuid.UUID]int),
		tripOfferTrip:      make(map[uuid.UUID]tripsModels.Trip),
		tripExcluded:       make(map[uuid.UUID]map[uuid.UUID]struct{}),
		tripWithdrawn:      make(map[uuid.UUID]struct{}),
		completingTrips:    make(map[uuid.UUID]struct{}),
		acceptingRiders:    make(map[uuid.UUID]struct{}),
		pausedRiders:       make(map[uuid.UUID]struct{}),
		repo:               repo,
		stream:             stream,
		bcv:                bcv,
		tripsRepo:          tripsRepo,
		tripEarnings:       tripEarnings,
		customers:          customers,
		balance:            balance,
		passengers:         passengers,
		vehicles:           vehicles,
		presence:           presence,
		riderTrips:         make(map[uuid.UUID]riderAssignment),
		offerTimeouts:      offerTimeouts,
		lastKnown:          lastKnown,
		push:               push,
	}
}

// riderTripOfferDTO es el viaje como lo ve el rider: fare_usd/fare_bs son su
// ganancia (RiderShare), igual que delivery_fee en los pedidos. collect_usd/
// collect_bs siguen siendo el total que cobra en efectivo. Bs siempre con la
// tasa congelada del viaje.
// pickupDistanceKm es la distancia de ESTE rider al pickup: en la oferta (antes de
// aceptar) es la suya propia dentro del matching; ya aceptado, es la congelada en
// trip.PickupDistanceKm. Sin sumar el bono acá, el rider solo vería el precio real
// hasta completar el viaje y seguiría rechazando ofertas lejanas por "no compensar".
func riderTripOfferDTO(trip *tripsModels.Trip, pickupDistanceKm float64) tripsDtos.TripResponse {
	frozen := *trip
	frozen.ApplyFrozenBsFare()
	resp := frozen.ToDTO()
	resp.DiscountUSD = 0
	if trip.Fare == nil {
		return resp
	}
	// El pago del rider sale de la tarifa completa: el descuento de primer viaje lo absorbe la plataforma.
	gross := trip.GrossFareUSD() + tripfare.PickupBonus(pickupDistanceKm)
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
// arriving e in_progress. Ya aceptado, PickupDistanceKm viene congelado en el trip.
func (r *appRidersManager) riderTripDTO(ctx context.Context, trip *tripsModels.Trip) tripsDtos.TripResponse {
	resp := riderTripOfferDTO(trip, trip.PickupDistanceKmOrZero())
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
// pickupDistanceKm es la de este candidato puntual: la misma oferta no se
// reusa entre candidatos porque cada uno está a su propia distancia del pickup.
func (r *appRidersManager) tripOfferWithPassenger(ctx context.Context, trip *tripsModels.Trip, pickupDistanceKm float64) tripsDtos.TripResponse {
	resp := riderTripOfferDTO(trip, pickupDistanceKm)
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
	r.mu.Unlock()

	r.restorePaused(ctx, riderID)

	vehicle := r.cacheRiderVehicle(ctx, riderID)

	// Un rider que se reconecta puede venir en otro modo: su ubicación ya
	// conocida tiene que quedar publicada en el índice del modo nuevo.
	r.mu.RLock()
	known, hadKnownLocation := r.riderLocation[riderID]
	r.mu.RUnlock()
	if hadKnownLocation {
		r.savePresence(ctx, riderID, known, mode, vehicle)
	}

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

// savePresence publica la ubicación del rider fuera del proceso. Un fallo de
// Redis no corta el flujo del socket: la memoria sigue siendo la fuente de verdad.
func (r *appRidersManager) savePresence(ctx context.Context, riderID uuid.UUID, location riderModels.RiderLocation, mode riderModels.RiderMode, vehicle string) {
	if r.presence == nil {
		return
	}
	if mode != riderModels.RiderModeTrips {
		mode = riderModels.RiderModeOrders
	}
	presence := riderModels.RiderPresence{
		RiderID:  riderID,
		Location: location,
		Mode:     mode,
		// Canónico: el filtro del adapter compara exacto y el perfil trae "Bici" tal cual lo escribió el staff.
		Vehicle: riderModels.NormalizeVehicleType(vehicle),
	}
	if err := r.presence.SaveLocation(ctx, presence); err != nil {
		slog.Error("error publicando la presencia del rider", "riderID", riderID, "error", err)
	}
	r.refreshPausedTTL(ctx, riderID)
	r.saveLastKnownLocation(riderID, location, mode, presence.Vehicle)
}

// refreshPausedTTL extiende la pausa junto con la presencia: sin esto, un rider
// pausado por más de PresenceTTL volvería a aparecer disponible solo porque
// sigue reportando ubicación.
func (r *appRidersManager) refreshPausedTTL(ctx context.Context, riderID uuid.UUID) {
	r.mu.RLock()
	_, paused := r.pausedRiders[riderID]
	r.mu.RUnlock()
	if !paused {
		return
	}
	if err := r.presence.SetPaused(ctx, riderID, true); err != nil {
		slog.Error("error refrescando la pausa del rider", "riderID", riderID, "error", err)
	}
}

// restorePaused releva en memoria el estado de pausa persistido en Redis al
// (re)conectar: una caída de red no debe obligar al rider a pausar de nuevo.
func (r *appRidersManager) restorePaused(ctx context.Context, riderID uuid.UUID) {
	if r.presence == nil {
		return
	}
	paused, err := r.presence.IsPaused(ctx, riderID)
	if err != nil {
		slog.Error("error leyendo la pausa del rider", "riderID", riderID, "error", err)
		return
	}
	r.mu.Lock()
	if paused {
		r.pausedRiders[riderID] = struct{}{}
	} else {
		delete(r.pausedRiders, riderID)
	}
	r.mu.Unlock()
}

// SetAvailability pausa o reanuda la recepción de nuevas órdenes y viajes. El
// rider sigue conectado y conserva lo que ya tiene asignado.
func (r *appRidersManager) SetAvailability(ctx context.Context, riderID uuid.UUID, acceptingNewWork bool) error {
	paused := !acceptingNewWork
	if r.presence == nil {
		return errcustom.ErrUnexpectedError
	}
	if err := r.presence.SetPaused(ctx, riderID, paused); err != nil {
		return err
	}
	r.mu.Lock()
	if paused {
		r.pausedRiders[riderID] = struct{}{}
	} else {
		delete(r.pausedRiders, riderID)
	}
	r.mu.Unlock()
	return nil
}

// saveLastKnownLocation persiste sin TTL lo mismo que savePresence manda a
// Redis, para que el matching tenga un respaldo cuando la presencia viva
// expira (rider sin la app abierta). Fire-and-forget: nunca debe frenar el
// loop de mensajes del WS por una escritura a Postgres.
func (r *appRidersManager) saveLastKnownLocation(riderID uuid.UUID, location riderModels.RiderLocation, mode riderModels.RiderMode, vehicle string) {
	if r.lastKnown == nil {
		return
	}
	go func() {
		// Un timeout acotado evita que esta goroutine quede viva indefinidamente
		// si Postgres se degrada — se dispara en cada update_location de cada
		// rider conectado, así que sin tope el volumen acumula goroutines rápido.
		ctx, cancel := context.WithTimeout(context.Background(), lastKnownLocationTimeout)
		defer cancel()
		loc := riderModels.RiderLastKnownLocation{
			RiderID:   riderID,
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
			Mode:      mode,
			Vehicle:   vehicle,
		}
		if err := r.lastKnown.Save(ctx, loc); err != nil {
			slog.Error("error guardando la última ubicación conocida del rider", "riderID", riderID, "error", err)
		}
	}()
}

// removePresence corre con su propio contexto: cuando se llega acá el socket ya
// se cerró y el contexto del request está cancelado.
func (r *appRidersManager) removePresence(riderID uuid.UUID) {
	if r.presence == nil {
		return
	}
	if err := r.presence.Remove(context.Background(), riderID); err != nil {
		slog.Error("error borrando la presencia del rider", "riderID", riderID, "error", err)
	}
}

func (r *appRidersManager) Unregister(riderID uuid.UUID, conn ridersPorts.RiderConn) {
	r.mu.Lock()
	stored, ok := r.availableRiders[riderID]
	disconnected := ok && stored == conn
	if disconnected {
		delete(r.availableRiders, riderID)
		delete(r.riderMode, riderID)
		delete(r.riderLocation, riderID)
		delete(r.riderVehicle, riderID)
		delete(r.pausedRiders, riderID)
	}
	r.mu.Unlock()

	if disconnected {
		r.removePresence(riderID)
	}

	conn.Close()
}

func (r *appRidersManager) UpdateLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64) error {
	r.mu.RLock()
	mode := r.riderMode[riderID]
	vehicle := r.riderVehicle[riderID]
	r.mu.RUnlock()
	return r.applyLocationUpdate(ctx, riderID, lat, lng, mode, vehicle)
}

// ReportLocation guarda la ubicación del rider fuera del WebSocket: la app la
// manda también en background, cuando el socket ya no está vivo (se congela o
// el OS lo mata). mode viene explícito del body porque sin conexión no hay de
// dónde inferirlo — a diferencia de update_location, que lo lee del rider ya
// registrado. vehicle reusa el caché en memoria si ya existe (de una conexión
// WS previa) y solo golpea el perfil cuando no hay nada cacheado — este
// endpoint puede llamarse con mucha más frecuencia que una conexión nueva.
func (r *appRidersManager) ReportLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64, mode riderModels.RiderMode) error {
	r.mu.RLock()
	vehicle, cached := r.riderVehicle[riderID]
	r.mu.RUnlock()
	if !cached {
		vehicle = r.cacheRiderVehicle(ctx, riderID)
	}
	return r.applyLocationUpdate(ctx, riderID, lat, lng, mode, vehicle)
}

// applyLocationUpdate es el núcleo que comparten update_location (WebSocket) y
// ReportLocation (background): valida y guarda la ubicación, publica presencia
// y última ubicación conocida, y notifica a quien tenga un viaje o pedido
// asignado. No depende de un socket vivo — offerOpenTripsToRider ya valida la
// conexión por su cuenta y es un no-op sin ella.
func (r *appRidersManager) applyLocationUpdate(ctx context.Context, riderID uuid.UUID, lat, lng float64, mode riderModels.RiderMode, vehicle string) error {
	// LatLngToCell acepta cualquier coordenada del planeta: el área de servicio se exige aparte.
	if !geo.InVenezuela(lat, lng) {
		return errcustom.ErrInvalidCoordinates
	}

	cell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	if err != nil {
		return errcustom.ErrInvalidCoordinates
	}

	location := riderModels.RiderLocation{
		Latitude:  lat,
		Longitude: lng,
		H3Cell:    cell,
		UpdatedAt: time.Now(),
	}

	r.mu.Lock()
	prev, hadLocation := r.riderLocation[riderID]
	r.riderLocation[riderID] = location
	r.riderMode[riderID] = mode
	inTripsMode := mode == riderModels.RiderModeTrips
	assignment, onTrip := r.riderTrips[riderID]
	r.mu.Unlock()

	r.savePresence(ctx, riderID, location, mode, vehicle)

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
	loc, hasLoc := r.riderLocation[riderID]
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
		pickupDistanceKm := 0.0
		if hasLoc {
			pickupDistanceKm = geo.HaversineKm(loc.Latitude, loc.Longitude, trip.PickupLatitude, trip.PickupLongitude)
		}
		data, err := json.Marshal(tripBroadcast{Type: "trip_offer", Trip: r.tripOfferWithPassenger(ctx, &trip, pickupDistanceKm)})
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
	riderIDs := make([]uuid.UUID, 0, len(r.availableRiders))
	seen := make(map[uuid.UUID]struct{}, len(r.availableRiders))
	for riderID, c := range r.availableRiders {
		if r.riderMode[riderID] == riderModels.RiderModeTrips {
			continue
		}
		if _, paused := r.pausedRiders[riderID]; paused {
			// Pausado pero conectado: no recibe el broadcast, y tampoco el
			// fallback de push por última ubicación conocida más abajo.
			seen[riderID] = struct{}{}
			continue
		}
		if !riderModels.VehicleCanCarryShopType(r.riderVehicle[riderID], shopType) {
			continue
		}
		clients = append(clients, c)
		riderIDs = append(riderIDs, riderID)
		seen[riderID] = struct{}{}
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.Send(data)
	}

	r.pushNewOrder(order, msg.Shop, shopType, riderIDs, seen)

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

// pushNewOrder manda el push a los riders conectados en modo pedidos (mismo
// pool que ya recibe el new_order por WS) más el respaldo de última ubicación
// conocida alrededor del shop, para riders sin presencia viva. Sin shop no
// hay centro para el respaldo geográfico: se pushea solo a los conectados.
// pushNewOrder nunca corre en el camino síncrono del llamador: BroadcastNewOrder
// se invoca directo desde el handler de cambio de estado de la orden, así que
// hasta el fallback por última ubicación conocida (consulta a Postgres) tiene
// que quedar en su propia goroutine — una consulta lenta no debe agregar
// latencia a la petición HTTP del shop.
func (r *appRidersManager) pushNewOrder(order ordersModels.Order, shop *ordersModels.ShopOrder, shopType string, riderIDs []uuid.UUID, exclude map[uuid.UUID]struct{}) {
	if r.push == nil {
		return
	}
	go func() {
		if shop != nil && r.lastKnown != nil {
			queryCtx, cancel := context.WithTimeout(context.Background(), lastKnownLocationTimeout)
			entries, err := r.lastKnown.SearchNearby(queryCtx, riderModels.LastKnownLocationQuery{
				Latitude:  shop.Latitude,
				Longitude: shop.Longitude,
				RadiusKm:  offlineOrderFallbackRadiusKm,
				Mode:      riderModels.RiderModeOrders,
				MaxAge:    riderModels.LastKnownLocationMaxAge,
				Exclude:   exclude,
			})
			cancel()
			if err != nil {
				slog.Error("error buscando riders por última ubicación conocida para new_order", "orderID", order.ID, "error", err)
			} else {
				for _, e := range entries {
					if !riderModels.VehicleCanCarryShopType(e.Vehicle, shopType) {
						continue
					}
					riderIDs = append(riderIDs, e.RiderID)
				}
			}
		}
		if len(riderIDs) == 0 {
			return
		}
		ctx := context.Background()
		r.push.NotifyNewOrder(ctx, riderIDs, order)
	}()
}

func (r *appRidersManager) BroadcastOrderUpdate(ctx context.Context, order ordersModels.Order) {
	data, err := json.Marshal(r.orderBroadcast(ctx, "order_update", order, true))
	if err != nil {
		slog.Error("error marshaling order_update for rider", "error", err)
		return
	}

	r.mu.Lock()
	var assignedRiderID *uuid.UUID
	for riderID, orderIDs := range r.assignedOrders {
		if !slices.Contains(orderIDs, order.ID) {
			continue
		}
		id := riderID
		assignedRiderID = &id
		if c, ok := r.availableRiders[riderID]; ok {
			c.Send(data)
		}
	}
	if isTerminalOrderStatus(order.Status) {
		r.forgetOrderLocked(order.ID)
	}
	r.mu.Unlock()

	// Solo Rejected/Cancelled son mala noticia para quien ya la tenía asignada:
	// Completed ocurre con el rider entregando, de seguro con la app abierta.
	status := ordersModels.OrderStatus(order.Status)
	if status == ordersModels.StatusRejected || status == ordersModels.StatusCancelled {
		r.pushOrderCancelled(order, assignedRiderID)
	}
}

// pushOrderCancelled corre en su propia goroutine por la misma razón que
// pushTripCancelled: no puede agregar latencia al camino síncrono del cambio
// de estado de la orden.
func (r *appRidersManager) pushOrderCancelled(order ordersModels.Order, riderID *uuid.UUID) {
	if r.push == nil || riderID == nil {
		return
	}
	go r.push.NotifyOrderCancelled(context.Background(), []uuid.UUID{*riderID}, order)
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
	// Las rondas sobreviven al request que pidió el viaje: siguen en timers.
	ctx := context.Background()
	maxRings := tripSearchMaxRings + round - 1
	exclusions := r.offerExclusions(trip.ID)
	candidates := r.nearestTripsRiders(ctx, trip, exclusions, maxRings)
	if len(candidates) == 0 {
		// Sin celda H3 el viaje nunca va a matchear: no tiene sentido esperar.
		if trip.PickupH3Cell == nil {
			r.expireTripIfStillSearching(trip.ID)
			return
		}
		// Riders sin presencia viva pero cerca por su última ubicación conocida:
		// no entran a la ronda en vivo (no hay a quién mandarle WS), pero sí
		// reciben el push — es justo el caso que esta ronda vacía no cubre.
		r.pushTripOfferToOfflineCandidates(trip, exclusions, maxRings)
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
		r.scheduleOfferTimeout(ctx, trip, tripEmptyRoundTimeout)
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
		conn, ok := r.availableRiders[c.RiderID]
		if !ok {
			continue
		}
		// Cada candidato ve su propio bono: la misma oferta no se reusa entre
		// riders porque cada uno está a su propia distancia del pickup.
		pickupDistanceKm := geo.HaversineKm(c.Location.Latitude, c.Location.Longitude, trip.PickupLatitude, trip.PickupLongitude)
		data, err := json.Marshal(tripBroadcast{Type: "trip_offer", Trip: r.tripOfferWithPassenger(ctx, &trip, pickupDistanceKm)})
		if err != nil {
			slog.Error("error marshaling trip_offer broadcast", "riderID", c.RiderID, "error", err)
			continue
		}
		conn.Send(data)
	}
	r.mu.RUnlock()

	r.pushTripOffer(trip, candidates, exclusions, maxRings)
	r.scheduleOfferTimeout(ctx, trip, tripAcceptTimeout)
}

// pushTripOffer manda el push a toda la ronda: los candidatos en vivo (hayan
// quedado conectados o no en el instante del WS send) más el respaldo de
// última ubicación conocida para riders sin presencia viva. El pool nunca se
// amplía más allá de lo que ya calcula el matching en vivo (mismo radio).
// Corre entero en su propia goroutine: offerTripRound se llama directo desde
// el handler de creación de viajes, y offlineTripCandidates pega contra
// Postgres — eso no puede agregar latencia a la petición del cliente.
func (r *appRidersManager) pushTripOffer(trip tripsModels.Trip, candidates []riderModels.RiderLocationEntry, exclusions map[uuid.UUID]struct{}, maxRings int) {
	if r.push == nil {
		return
	}
	go func() {
		ctx := context.Background()
		riderIDs := make([]uuid.UUID, 0, len(candidates))
		seen := make(map[uuid.UUID]struct{}, len(candidates))
		for _, c := range candidates {
			riderIDs = append(riderIDs, c.RiderID)
			seen[c.RiderID] = struct{}{}
		}
		riderIDs = append(riderIDs, r.offlineTripCandidates(ctx, trip, seen, exclusions, maxRings)...)
		r.push.NotifyTripOffer(ctx, riderIDs, trip)
	}()
}

// pushTripOfferToOfflineCandidates cubre la ronda vacía: no hay nadie en vivo
// para WS, pero puede haber riders cerca por última ubicación conocida.
// Misma razón que pushTripOffer para correr en su propia goroutine.
func (r *appRidersManager) pushTripOfferToOfflineCandidates(trip tripsModels.Trip, exclusions map[uuid.UUID]struct{}, maxRings int) {
	if r.push == nil {
		return
	}
	go func() {
		ctx := context.Background()
		riderIDs := r.offlineTripCandidates(ctx, trip, nil, exclusions, maxRings)
		if len(riderIDs) == 0 {
			return
		}
		r.push.NotifyTripOffer(ctx, riderIDs, trip)
	}()
}

// offlineTripCandidates busca en la última ubicación conocida (sin TTL) a los
// riders que SearchNearby no puede ver porque ya no tienen presencia viva en
// Redis. Mismo radio que la ronda en vivo, acotado por LastKnownLocationMaxAge.
func (r *appRidersManager) offlineTripCandidates(ctx context.Context, trip tripsModels.Trip, alreadyLive map[uuid.UUID]struct{}, exclusions map[uuid.UUID]struct{}, maxRings int) []uuid.UUID {
	if r.lastKnown == nil {
		return nil
	}
	exclude := make(map[uuid.UUID]struct{}, len(exclusions)+len(alreadyLive))
	for id := range exclusions {
		exclude[id] = struct{}{}
	}
	for id := range alreadyLive {
		exclude[id] = struct{}{}
	}
	queryCtx, cancel := context.WithTimeout(ctx, lastKnownLocationTimeout)
	defer cancel()
	entries, err := r.lastKnown.SearchNearby(queryCtx, riderModels.LastKnownLocationQuery{
		Latitude:        trip.PickupLatitude,
		Longitude:       trip.PickupLongitude,
		RadiusKm:        geo.RingsToKm(maxRings),
		Mode:            riderModels.RiderModeTrips,
		MaxAge:          riderModels.LastKnownLocationMaxAge,
		ExcludeVehicles: []string{riderModels.VehicleTypeBike},
		Exclude:         exclude,
	})
	if err != nil {
		slog.Error("error buscando riders por última ubicación conocida para oferta de viaje", "tripID", trip.ID, "error", err)
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	riderIDs := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		if r.riderBusyLocked(e.RiderID) {
			continue
		}
		riderIDs = append(riderIDs, e.RiderID)
	}
	return riderIDs
}

// scheduleOfferTimeout programa el vencimiento de la ronda en Redis en vez de
// un time.AfterFunc en memoria, para que un reinicio del proceso no lo pierda
// (ver #359). El TripOfferTimeoutWorker es quien lo dispara.
func (r *appRidersManager) scheduleOfferTimeout(ctx context.Context, trip tripsModels.Trip, timeout time.Duration) {
	r.mu.RLock()
	_, withdrawn := r.tripWithdrawn[trip.ID]
	r.mu.RUnlock()
	if withdrawn {
		return
	}
	if err := r.offerTimeouts.Schedule(ctx, trip.ID, time.Now().Add(timeout)); err != nil {
		slog.Error("error scheduling trip offer timeout", "tripID", trip.ID, "error", err)
	}
}

// onTripOfferTimeout corre cuando el TripOfferTimeoutWorker reclama un
// vencimiento ya cumplido. Relee el viaje desde Postgres en vez de confiar en
// un snapshot en memoria: por diseño, PickupH3Cell ya está garantizado no-nil
// acá (offerTripRound expira el viaje sin programar nada si viene nil).
func (r *appRidersManager) onTripOfferTimeout(ctx context.Context, tripID uuid.UUID) {
	current, err := r.tripsRepo.GetTripByID(ctx, tripID)
	if err != nil {
		slog.Error("error reading trip on offer timeout", "tripID", tripID, "error", err)
		return
	}
	if current.Status != tripsModels.TripStatusSearching {
		r.clearTripOfferState(ctx, current.ID)
		return
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
	for id := range r.pausedRiders {
		out[id] = struct{}{}
	}
	return out
}

// riderBusyLocked requiere r.mu tomado.
func (r *appRidersManager) riderBusyLocked(riderID uuid.UUID) bool {
	if _, onTrip := r.riderTrips[riderID]; onTrip {
		return true
	}
	if _, paused := r.pausedRiders[riderID]; paused {
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

	r.clearTripOfferState(ctx, tripID)
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

// publishTripStatusChanged avisa al ETL que CubeTrip debe resincronizarse.
// Las transiciones que lleva el rider (aceptar, avanzar, completar) viven
// acá y no en trips/app — ver "Trips & Riders" en CLAUDE.md — así que
// necesitan su propio punto de publicación, no pueden reusar publishTripEvent.
func (r *appRidersManager) publishTripStatusChanged(ctx context.Context, trip *tripsModels.Trip) {
	if r.stream == nil {
		return
	}
	event := tripsModels.DataEventTrip{}
	event.BuildFromTrip(tripsModels.EventTripStatusChanged, *trip)
	if err := r.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing trip_status_changed event", "tripID", trip.ID, "error", err)
	}
}

// notifyOtherCandidatesTripTaken avisa trip_taken a los demás riders que
// tenían la oferta viva de este viaje (no solo al que intenta aceptar y
// pierde la carrera, que ya lo recibe vía rejectAssignment) y limpia el
// estado de la ronda, igual que clearTripOfferState.
func (r *appRidersManager) notifyOtherCandidatesTripTaken(ctx context.Context, tripID, winnerID uuid.UUID) {
	msg, err := json.Marshal(map[string]string{"type": "trip_taken", "trip_id": tripID.String()})
	if err != nil {
		slog.Error("error marshaling trip_taken", "tripID", tripID, "error", err)
	}

	r.mu.Lock()
	recipients := make(map[uuid.UUID]ridersPorts.RiderConn)
	collect := func(riderID uuid.UUID) {
		if riderID == winnerID {
			return
		}
		if conn, ok := r.availableRiders[riderID]; ok {
			recipients[riderID] = conn
		}
	}
	for riderID := range r.tripOffers[tripID] {
		collect(riderID)
	}
	for riderID := range r.tripAlreadyOffered[tripID] {
		collect(riderID)
	}
	delete(r.tripOffers, tripID)
	delete(r.tripAlreadyOffered, tripID)
	delete(r.tripOfferRounds, tripID)
	delete(r.tripOfferTrip, tripID)
	delete(r.tripExcluded, tripID)
	r.mu.Unlock()

	if msg != nil {
		for _, conn := range recipients {
			conn.Send(msg)
		}
	}

	if err := r.offerTimeouts.Cancel(ctx, tripID); err != nil {
		slog.Error("error cancelling trip offer timeout", "tripID", tripID, "error", err)
	}
}

func (r *appRidersManager) clearTripOfferState(ctx context.Context, tripID uuid.UUID) {
	r.mu.Lock()
	delete(r.tripOffers, tripID)
	delete(r.tripAlreadyOffered, tripID)
	delete(r.tripOfferRounds, tripID)
	delete(r.tripOfferTrip, tripID)
	delete(r.tripExcluded, tripID)
	r.mu.Unlock()
	if err := r.offerTimeouts.Cancel(ctx, tripID); err != nil {
		slog.Error("error cancelling trip offer timeout", "tripID", tripID, "error", err)
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

// nearestTripsRiders devuelve los candidatos más cercanos al pickup. No escala
// anillos: GEOSEARCH ya ordena por distancia, así que los N primeros dentro del
// radio máximo son los mismos que encontraba el disco más chico con N dentro.
func (r *appRidersManager) nearestTripsRiders(ctx context.Context, trip tripsModels.Trip, exclude map[uuid.UUID]struct{}, maxRings int) []riderModels.RiderLocationEntry {
	// Sin celda H3 el viaje no entra al matching; offerTripRound lo da por perdido.
	if trip.PickupH3Cell == nil || r.presence == nil {
		return nil
	}

	entries, err := r.presence.SearchNearby(ctx, riderModels.PresenceQuery{
		Latitude:        trip.PickupLatitude,
		Longitude:       trip.PickupLongitude,
		RadiusKm:        geo.RingsToKm(maxRings),
		Mode:            riderModels.RiderModeTrips,
		MaxAge:          riderModels.TripMatchRiderMaxAge,
		ExcludeRiders:   exclude,
		ExcludeVehicles: []string{riderModels.VehicleTypeBike},
	})
	if err != nil {
		slog.Error("error buscando riders para el viaje", "tripID", trip.ID, "error", err)
		return nil
	}
	return r.connectedCandidates(entries)
}

// riderPickupDistanceKm devuelve 0 si la presencia del rider ya no está disponible al aceptar — sin bono, como antes de esta feature.
func (r *appRidersManager) riderPickupDistanceKm(ctx context.Context, trip tripsModels.Trip, riderID uuid.UUID) float64 {
	if r.presence == nil {
		return 0
	}
	// Radio amplio a propósito: cubre hasta la ronda más lejana en la que pudo haberse ofertado el viaje.
	maxRings := tripSearchMaxRings + tripOfferMaxRounds - 1
	entries, err := r.presence.SearchNearby(ctx, riderModels.PresenceQuery{
		Latitude:  trip.PickupLatitude,
		Longitude: trip.PickupLongitude,
		RadiusKm:  geo.RingsToKm(maxRings),
		Mode:      riderModels.RiderModeTrips,
		MaxAge:    riderModels.TripMatchRiderMaxAge,
	})
	if err != nil {
		slog.Error("error buscando ubicación del rider para el bono de acercamiento", "tripID", trip.ID, "riderID", riderID, "error", err)
		return 0
	}
	for _, entry := range entries {
		if entry.RiderID == riderID {
			return geo.HaversineKm(entry.Location.Latitude, entry.Location.Longitude, trip.PickupLatitude, trip.PickupLongitude)
		}
	}
	return 0
}

// connectedCandidates deja solo a los riders con socket en esta réplica y recién
// ahí aplica el tope. La presencia en Redis sobrevive al reinicio del proceso
// por su TTL, así que después de un deploy hay riders frescos que todavía no
// reconectaron: tomarlos como candidatos quema un cupo y los marca como ya
// ofertados sin que a nadie le llegue nada.
func (r *appRidersManager) connectedCandidates(entries []riderModels.RiderLocationEntry) []riderModels.RiderLocationEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]riderModels.RiderLocationEntry, 0, tripOfferCandidateCount)
	for _, entry := range entries {
		if _, connected := r.availableRiders[entry.RiderID]; !connected {
			continue
		}
		out = append(out, entry)
		if len(out) == tripOfferCandidateCount {
			return out
		}
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
	pickupDistanceKm := r.riderPickupDistanceKm(ctx, *trip, riderID)
	ok, err := r.tripsRepo.AssignTripRider(ctx, tripID, riderID, pickupCode, pickupDistanceKm)
	if err != nil {
		return err
	}
	if !ok {
		return r.rejectAssignment(ctx, tripID, conn)
	}

	r.notifyOtherCandidatesTripTaken(ctx, tripID, riderID)

	acceptedAt := time.Now()
	trip.RiderID = &riderID
	trip.Status = tripsModels.TripStatusAccepted
	trip.AcceptedAt = &acceptedAt
	trip.PickupCode = &pickupCode
	trip.PickupDistanceKm = &pickupDistanceKm
	r.rememberRiderTrip(riderID, trip.ID, trip.CustomerID)
	r.publishTripStatusChanged(ctx, trip)
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
	r.publishTripStatusChanged(ctx, trip)
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
	r.publishTripStatusChanged(ctx, trip)
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
	r.publishTripStatusChanged(ctx, trip)
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
	r.publishTripStatusChanged(ctx, trip)
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
		earnErr = r.tripEarnings.RecordTripEarning(ctx, *trip.RiderID, trip.ID, gross, trip.PickupDistanceKmOrZero(), r.bcvRate(ctx))
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
func (r *appRidersManager) NotifyTripCancelled(ctx context.Context, trip tripsModels.Trip) {
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
	assignedRiderID := trip.RiderID
	if assignedRiderID != nil {
		collect(*assignedRiderID)
		delete(r.riderTrips, *assignedRiderID)
	}
	r.tripWithdrawn[trip.ID] = struct{}{}
	delete(r.tripOffers, trip.ID)
	delete(r.tripAlreadyOffered, trip.ID)
	delete(r.tripOfferRounds, trip.ID)
	delete(r.tripOfferTrip, trip.ID)
	delete(r.tripExcluded, trip.ID)
	r.mu.Unlock()

	if err := r.offerTimeouts.Cancel(ctx, trip.ID); err != nil {
		slog.Error("error cancelling trip offer timeout", "tripID", trip.ID, "error", err)
	}

	for _, conn := range recipients {
		conn.Send(payload)
	}

	// Solo al rider ya asignado: quien apenas tenía la oferta no perdió nada
	// que requiera avisarle fuera del WS.
	r.pushTripCancelled(trip, assignedRiderID)
}

// pushTripCancelled corre en su propia goroutine por la misma razón que
// pushTripOffer: no puede agregar latencia al camino síncrono de CancelTrip.
func (r *appRidersManager) pushTripCancelled(trip tripsModels.Trip, riderID *uuid.UUID) {
	if r.push == nil || riderID == nil {
		return
	}
	go r.push.NotifyTripCancelled(context.Background(), []uuid.UUID{*riderID}, trip)
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

// AdvanceTripOfferTimeout corre el mismo camino que el vencimiento de ronda
// (tripAcceptTimeout o tripEmptyRoundTimeout), sin esperar al
// TripOfferTimeoutWorker. Los tests lo usan para pasar de ronda sin esperar
// el timeout real.
func AdvanceTripOfferTimeout(svc ridersPorts.RidersApp, trip tripsModels.Trip) {
	mgr := svc.(*appRidersManager)
	ctx := context.Background()
	if err := mgr.offerTimeouts.Cancel(ctx, trip.ID); err != nil {
		slog.Error("error cancelling trip offer timeout in test", "tripID", trip.ID, "error", err)
	}
	mgr.onTripOfferTimeout(ctx, trip.ID)
}

// TripSearchMaxRingsForTest expone el tope de anillos de la primera ronda, para
// que los tests ubiquen riders dentro o fuera del radio sin fijar el número.
const TripSearchMaxRingsForTest = tripSearchMaxRings

// TripOfferCandidateCountForTest expone cuántos candidatos entran por ronda.
const TripOfferCandidateCountForTest = tripOfferCandidateCount

// StopTripOfferTimerForTest cancela el vencimiento de oferta en cola para que
// no dispare después de que el test ya terminó.
func StopTripOfferTimerForTest(svc ridersPorts.RidersApp, tripID uuid.UUID) {
	mgr := svc.(*appRidersManager)
	if err := mgr.offerTimeouts.Cancel(context.Background(), tripID); err != nil {
		slog.Error("error cancelling trip offer timeout in test", "tripID", tripID, "error", err)
	}
}
