package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func cashTripRequest(tripID, quoteID uuid.UUID) dtos.CreateTripRequest {
	return dtos.CreateTripRequest{ID: tripID, QuoteID: quoteID, MethodPayment: models.MethodEfectivo}
}

func createdTrip(tripID, customerID uuid.UUID, quote *models.TripQuote) *models.Trip {
	fare, rate := quote.FareUSD, quote.BcvRate
	return &models.Trip{
		ID:            tripID,
		CustomerID:    customerID,
		Status:        models.TripStatusSearching,
		Fare:          &fare,
		BcvRate:       &rate,
		MethodPayment: models.MethodEfectivo,
	}
}

func TestRequestTrip_RetryWithSameIDReturnsExistingTripWithoutSideEffects(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(createdTrip(tripID, customerID, quote), nil)
	// Sin mock de bcv.GetRate: la respuesta usa la tasa congelada en el viaje.

	resp, err := d.svc.RequestTrip(context.Background(), customerID, cashTripRequest(tripID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, tripID, resp.ID)
	require.Equal(t, string(models.TripStatusSearching), resp.Status)
	require.Equal(t, money.RoundBs(quote.FareUSD*quote.BcvRate), *resp.FareBs)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
	d.matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
	d.balance.AssertNotCalled(t, "DebitForReference", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestTrip_RetryOfPagoMovilTripDoesNotUploadProofAgain(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	trip := createdTrip(tripID, customerID, quote)
	trip.Status = models.TripStatusPaymentReview
	trip.MethodPayment = models.MethodPagoMovil
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(trip, nil)

	req := paidTripRequest(quote.QuoteID)
	req.ID = tripID
	resp, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, tripID, resp.ID)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	d.reviews.AssertNotCalled(t, "CreateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestTrip_FirstAttemptUsesClientIDAsTripID(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.ID == tripID
	})).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	resp, err := d.svc.RequestTrip(context.Background(), customerID, cashTripRequest(tripID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, tripID, resp.ID)
}

func TestRequestTrip_IDOfAnotherCustomerIsRejected(t *testing.T) {
	d := newTripsDeps(t)
	customerID, ownerID, tripID := uuid.New(), uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(createdTrip(tripID, ownerID, quote), nil)

	resp, err := d.svc.RequestTrip(context.Background(), customerID, cashTripRequest(tripID, quote.QuoteID))

	require.ErrorIs(t, err, errcustom.ErrTripAlreadyExists)
	require.Equal(t, uuid.Nil, resp.ID)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_ConcurrentRetryReturnsTripCreatedByTheOtherAttempt(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	// El otro intento consumió la cotización y creó el viaje entre la primera búsqueda y el consumo.
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound).Once()
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(nil, errcustom.ErrQuoteNotFound)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(createdTrip(tripID, customerID, quote), nil)

	resp, err := d.svc.RequestTrip(context.Background(), customerID, cashTripRequest(tripID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, tripID, resp.ID)
	d.repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_DuplicateKeyReturnsExistingTrip(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound).Once()
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Return(errcustom.ErrTripAlreadyExists)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(createdTrip(tripID, customerID, quote), nil)

	resp, err := d.svc.RequestTrip(context.Background(), customerID, cashTripRequest(tripID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, tripID, resp.ID)
	d.matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_QuoteNotFoundWithUnusedIDKeepsError(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := d.svc.RequestTrip(context.Background(), customerID, cashTripRequest(tripID, quote.QuoteID))

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
}

func TestRequestTrip_FailureAfterCreateIsNotMaskedAsRetry(t *testing.T) {
	d := newTripsDeps(t)
	customerID, tripID := uuid.New(), uuid.New()
	quote := sampleTripQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	req := dtos.CreateTripRequest{ID: tripID, QuoteID: quote.QuoteID, MethodPayment: models.MethodBalance}

	d.repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound).Once()
	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Return(nil)
	d.balance.On("DebitForReference", mock.Anything, customerID, tripID, models.ReferenceTypeTrip, expectedBs).Return(errcustom.ErrInsufficientBalance)
	d.repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusSearching, models.TripStatusCancelled).Return(true, nil)
	d.repo.On("SaveTripCancelReason", mock.Anything, tripID, mock.Anything).Return(nil).Maybe()

	_, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
	d.repo.AssertNumberOfCalls(t, "GetTripByID", 1)
}
