package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTripsApp(t *testing.T) (ports.TripsApp, *mockstrips.TripsRepo, *mockstrips.TripRiderMatcher, *mockbcvports.BcvApp, *mockstrips.TripQuotesRepo) {
	t.Helper()
	repo := mockstrips.NewTripsRepo(t)
	matcher := mockstrips.NewTripRiderMatcher(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	return app.NewTripsApp(repo, matcher, bcv, quotes), repo, matcher, bcv, quotes
}

func mockBcvRate(bcv *mockbcvports.BcvApp, rate float64) {
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: rate}, nil)
}

func sampleFareEstimateRequest(customerID uuid.UUID) dtos.FareEstimateRequest {
	return dtos.FareEstimateRequest{
		CustomerID:       customerID,
		PickupAddress:    "Av. Libertador, Caracas",
		PickupLatitude:   10.4806,
		PickupLongitude:  -66.9036,
		DropoffAddress:   "CC Sambil, Caracas",
		DropoffLatitude:  10.4956,
		DropoffLongitude: -66.8517,
	}
}

func sampleTripQuote(customerID uuid.UUID) *models.TripQuote {
	now := time.Now()
	return &models.TripQuote{
		QuoteID:          uuid.New(),
		CustomerID:       customerID,
		PickupAddress:    "Av. Libertador, Caracas",
		PickupLatitude:   10.4806,
		PickupLongitude:  -66.9036,
		DropoffAddress:   "CC Sambil, Caracas",
		DropoffLatitude:  10.4956,
		DropoffLongitude: -66.8517,
		DistanceMeters:   1500,
		DurationMin:      6,
		FareUSD:          0.8,
		FareBs:           32,
		BcvRate:          40,
		CreatedAt:        now,
		ExpiresAt:        now.Add(models.TripQuoteTTL),
	}
}

func TestEstimateFare_SavesQuoteAndReturnsFare(t *testing.T) {
	svc, _, _, bcv, quotes := newTripsApp(t)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.MatchedBy(func(q *models.TripQuote) bool {
		return q.CustomerID == customerID &&
			q.PickupAddress == req.PickupAddress &&
			q.DropoffAddress == req.DropoffAddress &&
			q.FareUSD == deliveryfee.DevPlaceholderFee
	}), models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, resp.QuoteID)
	require.Equal(t, deliveryfee.DevPlaceholderFee, resp.FareUSD)
	require.Equal(t, deliveryfee.DevPlaceholderFee*40, resp.FareBs)
	require.InDelta(t, geo.HaversineKm(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)*1000, resp.DistanceMeters, 0.001)
	require.WithinDuration(t, time.Now().Add(models.TripQuoteTTL), resp.ExpiresAt, 2*time.Second)
	require.NotNil(t, saved)
	require.Equal(t, resp.QuoteID, saved.QuoteID)
}

func TestEstimateFare_FailsWhenBcvRateUnavailable(t *testing.T) {
	svc, _, _, bcv, quotes := newTripsApp(t)
	req := sampleFareEstimateRequest(uuid.New())
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down"))

	_, err := svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
	quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestTrip_UsesQuoteFareAndCoordinatesWithoutRefetchingBcvRate(t *testing.T) {
	svc, repo, matcher, _, quotes := newTripsApp(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	req := dtos.CreateTripRequest{QuoteID: quote.QuoteID}

	quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.CustomerID == customerID &&
			trip.PickupAddress == quote.PickupAddress &&
			trip.PickupLatitude == quote.PickupLatitude &&
			trip.DropoffLatitude == quote.DropoffLatitude &&
			trip.Fare != nil && *trip.Fare == quote.FareUSD
	})).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = uuid.New()
	}).Return(nil)
	repo.On("UpdateTripStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), models.TripStatusRequested, models.TripStatusSearching).Return(true, nil)
	matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()
	// A propósito no se mockea bcv.GetRate: si RequestTrip la llamara, haría panic.

	resp, err := svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.NotNil(t, resp.FareUSD)
	require.Equal(t, quote.FareUSD, *resp.FareUSD)
	require.NotNil(t, resp.FareBs)
	require.Equal(t, quote.FareUSD*quote.BcvRate, *resp.FareBs)
	repo.AssertExpectations(t)
	matcher.AssertExpectations(t)
	quotes.AssertExpectations(t)
}

func TestRequestTrip_QuoteNotFound(t *testing.T) {
	svc, repo, _, _, quotes := newTripsApp(t)
	customerID, quoteID := uuid.New(), uuid.New()

	quotes.On("GetAndConsume", mock.Anything, customerID, quoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{QuoteID: quoteID})

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
	repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}

// TestRequestTrip_QuoteScopedToRequester: la key de Redis incluye customerID,
// así que un quote_id ajeno a quien llama ni siquiera matchea (no hay forma
// de que el repo devuelva la quote de otro cliente); el repo simplemente no
// la encuentra, igual que si nunca hubiera existido.
func TestRequestTrip_QuoteScopedToRequester(t *testing.T) {
	svc, repo, _, _, quotes := newTripsApp(t)
	requesterID, someQuoteID := uuid.New(), uuid.New()

	quotes.On("GetAndConsume", mock.Anything, requesterID, someQuoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := svc.RequestTrip(context.Background(), requesterID, dtos.CreateTripRequest{QuoteID: someQuoteID})

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
	repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_SkipsMatcherWhenSearchingTransitionFails(t *testing.T) {
	svc, repo, matcher, _, quotes := newTripsApp(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)

	quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	repo.On("CreateTrip", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = uuid.New()
	}).Return(nil)
	repo.On("UpdateTripStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), models.TripStatusRequested, models.TripStatusSearching).Return(false, nil)

	_, err := svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{QuoteID: quote.QuoteID})

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
}

func TestGetTrip_ForbiddenWhenNotOwner(t *testing.T) {
	svc, repo, _, _, _ := newTripsApp(t)
	tripID, ownerID, requesterID := uuid.New(), uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: ownerID}, nil)

	_, err := svc.GetTrip(context.Background(), tripID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrTripForbidden)
}

func TestGetTrip_NotFound(t *testing.T) {
	svc, repo, _, _, _ := newTripsApp(t)
	tripID, requesterID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.GetTrip(context.Background(), tripID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestGetTripHistory_MapsAllTrips(t *testing.T) {
	svc, repo, _, bcv, _ := newTripsApp(t)
	customerID := uuid.New()

	repo.On("GetTripHistoryByCustomer", mock.Anything, customerID).Return([]models.Trip{
		{ID: uuid.New(), CustomerID: customerID, Status: models.TripStatusCompleted},
		{ID: uuid.New(), CustomerID: customerID, Status: models.TripStatusCancelled},
	}, nil)
	mockBcvRate(bcv, 40)

	resp, err := svc.GetTripHistory(context.Background(), customerID)

	require.NoError(t, err)
	require.Len(t, resp.Trips, 2)
}

func TestCancelTrip_ForbiddenWhenNotOwner(t *testing.T) {
	svc, repo, _, _, _ := newTripsApp(t)
	tripID, ownerID, requesterID := uuid.New(), uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: ownerID, Status: models.TripStatusRequested}, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrTripForbidden)
	repo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTrip_RejectsTerminalStatus(t *testing.T) {
	svc, repo, _, _, _ := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusCompleted}, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.ErrorIs(t, err, errcustom.ErrTripNotCancellable)
	repo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTrip_CASLossReturnsNotCancellable(t *testing.T) {
	svc, repo, _, _, _ := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusRequested}, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusRequested, models.TripStatusCancelled).Return(false, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.ErrorIs(t, err, errcustom.ErrTripNotCancellable)
}

func TestCancelTrip_Success(t *testing.T) {
	svc, repo, _, bcv, _ := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusSearching}, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusSearching, models.TripStatusCancelled).Return(true, nil)
	mockBcvRate(bcv, 40)

	resp, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusCancelled), resp.Status)
}

// TestQuoteFlow_EndToEndFareNeverDrifts: lo que EstimateFare congela es lo que RequestTrip usa después.
func TestQuoteFlow_EndToEndFareNeverDrifts(t *testing.T) {
	svc, repo, matcher, bcv, quotes := newTripsApp(t)
	customerID := uuid.New()
	estimateReq := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	estimate, err := svc.EstimateFare(context.Background(), estimateReq)
	require.NoError(t, err)
	require.NotNil(t, saved)

	quotes.On("GetAndConsume", mock.Anything, customerID, estimate.QuoteID).Return(saved, nil)
	repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.Fare != nil && *trip.Fare == estimate.FareUSD
	})).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = uuid.New()
	}).Return(nil)
	repo.On("UpdateTripStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), models.TripStatusRequested, models.TripStatusSearching).Return(true, nil)
	matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	resp, err := svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{QuoteID: estimate.QuoteID})

	require.NoError(t, err)
	require.NotNil(t, resp.FareUSD)
	require.Equal(t, estimate.FareUSD, *resp.FareUSD)
	require.NotNil(t, resp.FareBs)
	require.Equal(t, estimate.FareBs, *resp.FareBs)
}
