package app_test

import (
	"context"
	"sync"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// newOpenTripSvc deja un viaje en searching que ningún rider vio todavía
// (ronda 1 sin candidatos), para probar qué recibe un rider que llega tarde.
func newOpenTripSvc(t *testing.T) (ridersPorts.RidersApp, *mockstrips.TripsRepo, tripsModels.Trip) {
	t.Helper()
	svc, tripsRepo, repo := newTripOfferSvc(t)
	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	// Maybe: en los casos negativos justo se espera que no se consulte el viaje.
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil).Maybe()

	asMatcher(t, svc).OfferTrip(context.Background(), trip)
	return svc, tripsRepo, trip
}

func registerWatchedRider(t *testing.T, svc ridersPorts.RidersApp, mode riderModels.RiderMode) (uuid.UUID, func() int) {
	t.Helper()
	offers, watch := countTripOffers(t)
	id := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	watch(id, conn)
	require.NoError(t, svc.Register(context.Background(), id, conn, mode))
	return id, func() int { return offers[id] }
}

func TestUpdateLocation_OffersOpenTripToLateRider(t *testing.T) {
	svc, tripsRepo, trip := newOpenTripSvc(t)
	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeTrips)

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4807, -66.9036))

	require.Equal(t, 1, offersReceived())

	// La oferta reenviada debe ser aceptable: el rider quedó dentro del set de ofertados.
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything).Return(true, nil)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return().Maybe()
	require.NoError(t, svc.AcceptTrip(context.Background(), riderID, trip.ID, conn))
	tripsRepo.AssertCalled(t, "AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything)
}

func TestUpdateLocation_DoesNotResendWhileStayingInSameCell(t *testing.T) {
	svc, _, _ := newOpenTripSvc(t)
	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeTrips)

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4807, -66.9036))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4807, -66.9036))

	require.Equal(t, 1, offersReceived())
}

func TestUpdateLocation_ResendsToRiderInCurrentRoundWhenCellChanges(t *testing.T) {
	svc, _, trip := newOpenTripSvc(t)
	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeTrips)

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4807, -66.9036))
	nextLat, nextLng := latLngAtRing(t, *trip.PickupH3Cell, 1)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, nextLat, nextLng))

	require.Equal(t, 2, offersReceived())
}

func TestUpdateLocation_DoesNotOfferOutsideCurrentRoundRadius(t *testing.T) {
	svc, _, trip := newOpenTripSvc(t)
	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeTrips)

	// Un anillo más allá del tope de la primera ronda: fuera del radio sea cual
	// sea el valor de la constante.
	farLat, farLng := latLngAtRing(t, *trip.PickupH3Cell, app.TripSearchMaxRingsForTest+1)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, farLat, farLng))

	require.Zero(t, offersReceived())
}

// El borde de adentro: el último anillo que la primera ronda alcanza sí oferta.
// Junto con el test de arriba fija el radio en el valor de la constante.
func TestUpdateLocation_OffersAtMaxRingOfFirstRound(t *testing.T) {
	svc, _, trip := newOpenTripSvc(t)
	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeTrips)

	lat, lng := latLngAtRing(t, *trip.PickupH3Cell, app.TripSearchMaxRingsForTest)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, lat, lng))

	require.Equal(t, 1, offersReceived())
}

func TestUpdateLocation_DoesNotOfferToRidersInOrdersMode(t *testing.T) {
	svc, _, _ := newOpenTripSvc(t)
	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeOrders)

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4807, -66.9036))

	require.Zero(t, offersReceived())
}

func TestUpdateLocation_DoesNotOfferTripThatIsNoLongerSearching(t *testing.T) {
	svc, tripsRepo, repo := newTripOfferSvc(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	cancelled := trip
	cancelled.Status = tripsModels.TripStatusCancelled
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&cancelled, nil)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	riderID, offersReceived := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4807, -66.9036))

	require.Zero(t, offersReceived())
}

func TestUpdateLocation_DoesNotOfferTripToRiderWhoCancelledIt(t *testing.T) {
	svc, tripsRepo, repo := newTripOfferSvc(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderA := uuid.New()
	offers, watch := countTripOffers(t)
	connA := mocksriders.NewRiderConn(t)
	watch(riderA, connA)
	require.NoError(t, svc.Register(context.Background(), riderA, connA, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderA, 10.4807, -66.9036))

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	trip.RiderID = &riderA
	trip.Status = tripsModels.TripStatusAccepted
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&trip, nil)
	tripsRepo.On("ReleaseTripRider", mock.Anything, trip.ID, riderA).Return(true, nil)
	require.NoError(t, svc.CancelAcceptedTrip(context.Background(), riderA, trip.ID, connA))

	nextLat, nextLng := latLngAtRing(t, *trip.PickupH3Cell, 1)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderA, nextLat, nextLng))

	require.Zero(t, offers[riderA])
}

func TestSendActiveTrips_ResendsAssignedTripWithoutCodesAndRelinksLocation(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	customer := &recordingCustomer{}
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, customer, nil, nil, nil, newPresenceStore(t))

	riderID := uuid.New()
	customerID := uuid.New()
	pickupCode := "1234"
	fare := 2.5
	frozenRate := 40.0
	trip := tripsModels.Trip{
		ID:         uuid.New(),
		CustomerID: customerID,
		RiderID:    &riderID,
		Status:     tripsModels.TripStatusAccepted,
		Fare:       &fare,
		BcvRate:    &frozenRate,
		PickupCode: &pickupCode,
	}
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return([]tripsModels.Trip{trip}, nil)

	var mu sync.Mutex
	var sent [][]byte
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, append([]byte(nil), args.Get(0).([]byte)...))
	}).Return()

	require.NoError(t, svc.SendActiveTrips(context.Background(), riderID, conn))

	require.Len(t, sent, 1)
	require.Equal(t, "trip_status", messageType(t, sent[0]))
	require.NotContains(t, string(sent[0]), "pickup_code")
	require.Contains(t, string(sent[0]), `"fare_usd":2.25,"fare_bs":90`)

	// El cliente vuelve a recibir la ubicación del conductor tras reconectar.
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))
	require.Equal(t, 1, customer.locations)
	require.Equal(t, trip.ID, customer.lastTrip)
}

func TestSendActiveTrips_NoActiveTripsSendsNothing(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t))

	riderID := uuid.New()
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return([]tripsModels.Trip{}, nil)
	conn := mocksriders.NewRiderConn(t)

	require.NoError(t, svc.SendActiveTrips(context.Background(), riderID, conn))

	conn.AssertNotCalled(t, "Send", mock.Anything)
	bcv.AssertNotCalled(t, "GetRate", mock.Anything)
}

func TestSendActiveTrips_PropagatesRepoError(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t))

	riderID := uuid.New()
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return(nil, errcustom.ErrUnexpectedError)

	err := svc.SendActiveTrips(context.Background(), riderID, mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestActiveTrips_ReturnsTripsWithBsFareAndNoCodes(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t))

	riderID := uuid.New()
	pickupCode, dropoffCode := "1234", "5678"
	fare := 2.5
	frozenRate := 40.0
	trip := tripsModels.Trip{
		ID:          uuid.New(),
		RiderID:     &riderID,
		Status:      tripsModels.TripStatusInProgress,
		Fare:        &fare,
		BcvRate:     &frozenRate,
		PickupCode:  &pickupCode,
		DropoffCode: &dropoffCode,
	}
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return([]tripsModels.Trip{trip}, nil)

	out, err := svc.ActiveTrips(context.Background(), riderID)

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, trip.ID, out[0].ID)
	require.Equal(t, "in_progress", out[0].Status)
	require.NotNil(t, out[0].FareBs)
	require.InDelta(t, 90, *out[0].FareBs, 0.001)
	require.Nil(t, out[0].PickupCode)
	require.Nil(t, out[0].DropoffCode)
}

func TestActiveTrips_EmptyReturnsEmptyListWithoutQueryingBcv(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t))

	riderID := uuid.New()
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return([]tripsModels.Trip{}, nil)

	out, err := svc.ActiveTrips(context.Background(), riderID)

	require.NoError(t, err)
	require.NotNil(t, out)
	require.Empty(t, out)
	bcv.AssertNotCalled(t, "GetRate", mock.Anything)
}

func TestActiveTrips_PropagatesRepoError(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t))

	riderID := uuid.New()
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return(nil, errcustom.ErrUnexpectedError)

	out, err := svc.ActiveTrips(context.Background(), riderID)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	require.Nil(t, out)
}
