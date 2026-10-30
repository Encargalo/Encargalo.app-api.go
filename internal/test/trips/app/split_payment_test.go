package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRequestTrip_CashAndBalanceOffersOnlyCashPortion(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(100.0, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.Status == models.TripStatusSearching &&
			trip.BalancePaidBs != nil && *trip.BalancePaidBs == 10 &&
			trip.CashAmountBs != nil && *trip.CashAmountBs == 20
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Trip).ID = uuid.New()
	}).Return(nil)
	d.balance.On("DebitForReference", mock.Anything, customerID, mock.AnythingOfType("uuid.UUID"), models.ReferenceTypeTrip, 10.0).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	resp, err := d.svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{
		QuoteID:         quote.QuoteID,
		MethodPayment:   models.MethodEfectivo,
		BalanceAmountBs: 12,
	})

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusSearching), resp.Status)
	require.NotNil(t, resp.CollectUSD)
	require.InDelta(t, 0.5, *resp.CollectUSD, 0.001)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
}
