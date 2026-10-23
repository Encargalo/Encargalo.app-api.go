package app_test

import (
	"context"
	"errors"
	"testing"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEstimateFare_FirstTripGetsHalfDiscount(t *testing.T) {
	svc, repo, _, bcv, quotes := newTripsApp(t)
	customerID := uuid.New()
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(false, nil)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), sampleFareEstimateRequest(customerID))

	require.NoError(t, err)
	discount := models.FirstTripDiscountUSD(tripfare.DevPlaceholderFee)
	require.Equal(t, tripfare.DevPlaceholderFee-discount, resp.FareUSD)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareOriginalUSD)
	require.Equal(t, discount, resp.DiscountUSD)
	require.Equal(t, resp.FareUSD*40, resp.FareBs)
	require.NotNil(t, saved)
	require.Equal(t, resp.FareUSD, saved.FareUSD, "la quote congela lo que paga el cliente")
	require.Equal(t, discount, saved.DiscountUSD)
}

func TestEstimateFare_CustomerWithTripsPaysFullFare(t *testing.T) {
	svc, repo, _, bcv, quotes := newTripsApp(t)
	customerID := uuid.New()
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), sampleFareEstimateRequest(customerID))

	require.NoError(t, err)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
	require.Zero(t, resp.DiscountUSD)
	require.Zero(t, resp.FareOriginalUSD)
}

// Si no se puede saber si es el primer viaje, se cotiza sin descuento en vez de fallar.
func TestEstimateFare_LookupFailureQuotesWithoutDiscount(t *testing.T) {
	svc, repo, _, bcv, quotes := newTripsApp(t)
	customerID := uuid.New()
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(false, errors.New("db caída"))
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), sampleFareEstimateRequest(customerID))

	require.NoError(t, err)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
	require.Zero(t, resp.DiscountUSD)
}

func TestRequestTrip_CopiesQuoteDiscountToTrip(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	quote.FareUSD = 4
	quote.DiscountUSD = 4

	var created *models.Trip
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		created = args.Get(1).(*models.Trip)
	}).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	resp, err := d.svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{
		QuoteID:       quote.QuoteID,
		MethodPayment: models.MethodEfectivo,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, 4.0, *created.Fare)
	require.Equal(t, 4.0, created.DiscountUSD)
	require.Equal(t, 4.0, resp.DiscountUSD)
}

// Dos cotizaciones con descuento: el índice único deja pasar solo un viaje.
func TestRequestTrip_SecondDiscountedTripIsRejected(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	quote.DiscountUSD = 0.8

	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Return(errcustom.ErrFirstTripDiscountUsed)

	_, err := d.svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{
		QuoteID:       quote.QuoteID,
		MethodPayment: models.MethodEfectivo,
	})

	require.ErrorIs(t, err, errcustom.ErrFirstTripDiscountUsed)
}
