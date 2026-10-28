package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestOfferTripRound_SchedulesTimeoutInStore cubre la mitad que #359 arregla:
// el vencimiento de la ronda se escribe en el store compartido (Redis en
// producción), no en un timer del proceso. Sin candidatos conectados entra
// por la rama de "ronda vacía" (tripEmptyRoundTimeout).
func TestOfferTripRound_SchedulesTimeoutInStore(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	offerTimeouts := mocksriders.NewTripOfferTimeoutStore(t)

	trip := samplePickupTrip(t)
	offerTimeouts.On("Schedule", mock.Anything, trip.ID, mock.AnythingOfType("time.Time")).Return(nil).Once()

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), offerTimeouts, nil, nil)
	matcher := asMatcher(t, svc)

	matcher.OfferTrip(context.Background(), trip)

	offerTimeouts.AssertExpectations(t)
}

// TestAcceptTrip_CancelsTimeoutInStore cubre la otra mitad: al aceptarse el
// viaje, el vencimiento pendiente se saca del store para que el
// TripOfferTimeoutWorker no dispare una ronda sobre un viaje que ya no está
// buscando.
func TestAcceptTrip_CancelsTimeoutInStore(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	offerTimeouts := mocksriders.NewTripOfferTimeoutStore(t)
	offerTimeouts.On("Schedule", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	offerTimeouts.On("Cancel", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), offerTimeouts, nil, nil)
	matcher := asMatcher(t, svc)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusSearching}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything).Return(true, nil)
	matcher.OfferTrip(context.Background(), trip)

	err := svc.AcceptTrip(context.Background(), riderID, trip.ID, conn)

	require.NoError(t, err)
	offerTimeouts.AssertExpectations(t)
}
