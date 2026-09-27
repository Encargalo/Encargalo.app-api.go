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

func TestRequestTrip_CashStartsSearchingAndOffersRider(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)

	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.Status == models.TripStatusSearching &&
			trip.MethodPayment == models.MethodEfectivo &&
			trip.Fare != nil && *trip.Fare == quote.FareUSD
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Trip).ID = uuid.New()
	}).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	resp, err := d.svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{
		QuoteID:       quote.QuoteID,
		MethodPayment: models.MethodEfectivo,
	})

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusSearching), resp.Status)
	require.Equal(t, models.MethodEfectivo, resp.MethodPayment)
	require.NotNil(t, resp.CollectUSD)
	require.Equal(t, quote.FareUSD, *resp.CollectUSD)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	d.reviews.AssertNotCalled(t, "CreateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
