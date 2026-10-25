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

func TestOfferTrip_CashTripSendsBsAtFrozenRate(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	var offers [][]byte
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		raw := args.Get(0).([]byte)
		if messageType(t, raw) == "trip_offer" {
			offers = append(offers, append([]byte(nil), raw...))
		}
	}).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	fare, rate := 2.5, 40.0
	trip.Fare = &fare
	trip.BcvRate = &rate
	trip.MethodPayment = tripsModels.MethodEfectivo
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	require.NotEmpty(t, offers)
	for _, msg := range offers {
		require.Contains(t, string(msg), `"fare_usd":2.25,"fare_bs":90`)
		require.Contains(t, string(msg), `"collect_usd":2.5,"collect_bs":100`)
	}
	bcv.AssertNotCalled(t, "GetRate", mock.Anything)
}
