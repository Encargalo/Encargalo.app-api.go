package app

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// TestTripOfferTimeoutWorker_ProcessDue_AdvancesEachClaimedTrip prueba el
// lado del worker que reemplaza el time.AfterFunc: cada trip_id que PopDue
// reclama se procesa (relee el viaje y sigue su ciclo), sin pasar por Redis
// real — eso ya lo cubre internal/test/riders/repo/offer_timeout_test.go.
func TestTripOfferTimeoutWorker_ProcessDue_AdvancesEachClaimedTrip(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mocksriders.NewTripsStore(t)
	earnings := mocksriders.NewTripEarningRecorder(t)
	offerTimeouts := mocksriders.NewTripOfferTimeoutStore(t)

	tripA, tripB := uuid.New(), uuid.New()
	offerTimeouts.On("PopDue", mock.Anything, int64(tripOfferTimeoutBatchSize)).Return([]uuid.UUID{tripA, tripB}, nil).Once()
	// Ya no están en "searching": el worker debe limitarse a limpiar su
	// estado de oferta, no debe reprogramar ni expirar nada.
	tripsRepo.On("GetTripByID", mock.Anything, tripA).Return(&tripsModels.Trip{ID: tripA, Status: tripsModels.TripStatusAccepted}, nil).Once()
	tripsRepo.On("GetTripByID", mock.Anything, tripB).Return(&tripsModels.Trip{ID: tripB, Status: tripsModels.TripStatusCancelled}, nil).Once()
	offerTimeouts.On("Cancel", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)

	svc := New(repo, stream, bcv, tripsRepo, earnings, nil, nil, nil, nil, nil, offerTimeouts, nil, nil)
	worker := NewTripOfferTimeoutWorker(svc, offerTimeouts, tripsRepo)

	worker.processDue(context.Background())

	offerTimeouts.AssertExpectations(t)
	tripsRepo.AssertExpectations(t)
}

// TestTripOfferTimeoutWorker_RescueOrphans_SchedulesOnlyAbsentOnes es el
// barrido de rescate que pide #359: programa un vencimiento inmediato para
// cada viaje en "searching", dejando que ScheduleIfAbsent (ZADD NX) decida
// si ya tenía uno.
func TestTripOfferTimeoutWorker_RescueOrphans_SchedulesOnlyAbsentOnes(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mocksriders.NewTripsStore(t)
	earnings := mocksriders.NewTripEarningRecorder(t)
	offerTimeouts := mocksriders.NewTripOfferTimeoutStore(t)

	orphan, withTimer := uuid.New(), uuid.New()
	tripsRepo.On("ListSearchingTripIDs", mock.Anything).Return([]uuid.UUID{orphan, withTimer}, nil).Once()
	offerTimeouts.On("ScheduleIfAbsent", mock.Anything, orphan, mock.AnythingOfType("time.Time")).Return(nil).Once()
	offerTimeouts.On("ScheduleIfAbsent", mock.Anything, withTimer, mock.AnythingOfType("time.Time")).Return(nil).Once()

	svc := New(repo, stream, bcv, tripsRepo, earnings, nil, nil, nil, nil, nil, offerTimeouts, nil, nil)
	worker := NewTripOfferTimeoutWorker(svc, offerTimeouts, tripsRepo)

	worker.rescueOrphans(context.Background())

	offerTimeouts.AssertExpectations(t)
	tripsRepo.AssertExpectations(t)
}
