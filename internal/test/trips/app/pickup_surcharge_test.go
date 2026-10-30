package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTripsAppWithPickupCandidates(t *testing.T) (*mockstrips.TripsRepo, *mockbcvports.BcvApp, *mockstrips.TripQuotesRepo, *mockstrips.PickupCandidatesLookup, ports.TripsApp) {
	t.Helper()
	repo := mockstrips.NewTripsRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	pickupCandidates := mockstrips.NewPickupCandidatesLookup(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, pickupCandidates)
	return repo, bcv, quotes, pickupCandidates, svc
}

// Caso real: rider a 4.54 km del pickup — cargo fijo de $0.80 (2 km × $0.40).
func TestEstimateFare_PickupSurcharge_AppliesWhenAvgAboveThreshold(t *testing.T) {
	repo, bcv, quotes, pickupCandidates, svc := newTripsAppWithPickupCandidates(t)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	pickupCandidates.On("AvgPickupDistanceKm", mock.Anything, req.PickupLatitude, req.PickupLongitude).Return(4.54, true)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, 0.80, resp.PickupSurchargeUSD)
	require.Equal(t, tripfare.DevPlaceholderFee+0.80, resp.FareUSD)
	require.Equal(t, 0.80, saved.PickupSurchargeUSD)
}

func TestEstimateFare_PickupSurcharge_NoneWhenAvgAtOrBelowThreshold(t *testing.T) {
	repo, bcv, quotes, pickupCandidates, svc := newTripsAppWithPickupCandidates(t)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	pickupCandidates.On("AvgPickupDistanceKm", mock.Anything, req.PickupLatitude, req.PickupLongitude).Return(2.0, true)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Zero(t, resp.PickupSurchargeUSD)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
}

// Sin ningún candidato disponible no se puede estimar: cotiza sin cargo en vez de bloquear.
func TestEstimateFare_PickupSurcharge_NoneWhenNoCandidateFound(t *testing.T) {
	repo, bcv, quotes, pickupCandidates, svc := newTripsAppWithPickupCandidates(t)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	pickupCandidates.On("AvgPickupDistanceKm", mock.Anything, req.PickupLatitude, req.PickupLongitude).Return(0.0, false)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Zero(t, resp.PickupSurchargeUSD)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
}

// Sin el lookup conectado (nil), igual que antes de esta feature: cotiza sin cargo.
func TestEstimateFare_PickupSurcharge_NilLookupSkipsSurcharge(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Zero(t, resp.PickupSurchargeUSD)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
}
