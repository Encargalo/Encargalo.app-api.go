package app_test

import (
	"context"
	"encoding/json"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Viaje de $8 con $4 de descuento pagado en efectivo: el rider cobra $4 y su pago es $7.20.
func TestAdvanceTrip_DiscountedCashTripCreditsRiderTheDifference(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	var sent tripsDtos.TripResponse
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		var msg struct {
			Trip tripsDtos.TripResponse `json:"trip"`
		}
		require.NoError(t, json.Unmarshal(args.Get(0).([]byte), &msg))
		sent = msg.Trip
	}).Return()

	fare, rate := 4.0, 40.0
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{
		ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusInProgress,
		Fare: &fare, DiscountUSD: 4, BcvRate: &rate, MethodPayment: tripsModels.MethodEfectivo,
	}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, fare, "5678").Return(true, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripCashCollected", mock.Anything, riderID, tripID, 8.0, 4.0, 40.0).Return(nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, "5678", conn)

	require.NoError(t, err)
	require.NotNil(t, sent.FareUSD)
	require.Equal(t, 7.2, *sent.FareUSD, "el rider ve su pago sobre la tarifa completa")
	require.NotNil(t, sent.CollectUSD)
	require.Equal(t, 4.0, *sent.CollectUSD, "cobra solo lo que paga el cliente")
	require.Zero(t, sent.DiscountUSD)
}

func TestAdvanceTrip_DiscountedBankTripPaysOnGrossFare(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	fare := 14.0
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{
		ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusInProgress,
		Fare: &fare, DiscountUSD: 6, MethodPayment: tripsModels.MethodBalance,
	}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, fare, "5678").Return(true, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripEarning", mock.Anything, riderID, tripID, 20.0, 40.0).Return(nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, "5678", conn)

	require.NoError(t, err)
}

func TestRecordTripCashCollected_CreditsWhenCashIsBelowRiderPay(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, nil, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	// Pago $7.20 - efectivo $4 = $3.20 a favor del rider.
	repo.On("RecordMovement", mock.Anything, riderID, "rider", 115.2, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeTripCash) && tx.TripID != nil && *tx.TripID == tripID
	})).Return(0.0, nil)

	err := svc.RecordTripCashCollected(context.Background(), riderID, tripID, 8, 4, 36)

	require.NoError(t, err)
}
