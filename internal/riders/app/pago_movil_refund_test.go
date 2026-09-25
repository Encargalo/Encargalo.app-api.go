package app

import (
	"testing"
	"time"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestExpireTripIfStillSearching_CreditsPagoMovil(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mocksriders.NewTripsStore(t)
	earnings := mocksriders.NewTripEarningRecorder(t)
	balance := mockpayments.NewCustomerBalanceCreditor(t)
	mgr := New(repo, stream, bcv, tripsRepo, earnings, nil, balance, nil).(*appRidersManager)

	fare := 5.0
	rate := 40.0
	tripID := uuid.New()
	customerID := uuid.New()
	txID := uuid.New()
	trip := &tripsModels.Trip{
		ID: tripID, CustomerID: customerID, Status: tripsModels.TripStatusNoDriver,
		Fare: &fare, BcvRate: &rate,
	}

	tripsRepo.On("UpdateTripStatus", mock.Anything, tripID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver).Return(true, nil)
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(trip, nil)
	balance.On("CreditOrderRejected", mock.Anything, customerID, tripID, tripsModels.ReferenceTypeTrip, 200.0, 200.0, 40.0).
		Return(txID, time.Now(), nil)
	tripsRepo.On("SaveTripPayment", mock.Anything, tripID, mock.Anything).Return(nil)

	mgr.expireTripIfStillSearching(tripID)

	balance.AssertCalled(t, "CreditOrderRejected", mock.Anything, customerID, tripID, tripsModels.ReferenceTypeTrip, 200.0, 200.0, 40.0)
}
