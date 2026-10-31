package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRequestShipment_RetryWithSameIDReturnsExistingShipmentWithoutSideEffects(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(createdShipment(shipmentID, customerID, quote), nil)
	// Sin mock de bcv.GetRate: la respuesta usa la tasa congelada en el envío.

	resp, err := d.svc.RequestShipment(context.Background(), customerID, balanceShipmentRequest(shipmentID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, shipmentID, resp.ID)
	require.Equal(t, string(models.ShipmentStatusSearching), resp.Status)
	require.Equal(t, money.RoundBs(quote.FareUSD*quote.BcvRate), *resp.FareBs)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateShipment", mock.Anything, mock.Anything)
	d.matcher.AssertNotCalled(t, "OfferShipment", mock.Anything, mock.Anything)
	d.balance.AssertNotCalled(t, "DebitForReference", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestShipment_RetryOfPagoMovilShipmentDoesNotUploadProofAgain(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	shipment := createdShipment(shipmentID, customerID, quote)
	shipment.Status = models.ShipmentStatusPaymentReview
	shipment.MethodPayment = models.MethodPagoMovil
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(shipment, nil)

	req := paidShipmentRequest(quote.QuoteID)
	req.ID = shipmentID
	resp, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, shipmentID, resp.ID)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	d.reviews.AssertNotCalled(t, "CreateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestShipment_FirstAttemptUsesClientIDAsShipmentID(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(nil, errcustom.ErrNotFound)
	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateShipment", mock.Anything, mock.MatchedBy(func(s *models.Shipment) bool {
		return s.ID == shipmentID
	})).Return(nil)
	d.balance.On("DebitForReference", mock.Anything, customerID, shipmentID, models.ReferenceTypeShipment, expectedBs).Return(nil)
	d.matcher.On("OfferShipment", mock.Anything, mock.Anything).Return()

	resp, err := d.svc.RequestShipment(context.Background(), customerID, balanceShipmentRequest(shipmentID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, shipmentID, resp.ID)
}

func TestRequestShipment_IDOfAnotherCustomerIsRejected(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, ownerID, shipmentID := uuid.New(), uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(createdShipment(shipmentID, ownerID, quote), nil)

	resp, err := d.svc.RequestShipment(context.Background(), customerID, balanceShipmentRequest(shipmentID, quote.QuoteID))

	require.ErrorIs(t, err, errcustom.ErrShipmentAlreadyExists)
	require.Equal(t, uuid.Nil, resp.ID)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateShipment", mock.Anything, mock.Anything)
}

func TestRequestShipment_ConcurrentRetryReturnsShipmentCreatedByTheOtherAttempt(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	// El otro intento consumió la cotización y creó el envío entre la primera búsqueda y el consumo.
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(nil, errcustom.ErrNotFound).Once()
	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(nil, errcustom.ErrQuoteNotFound)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(createdShipment(shipmentID, customerID, quote), nil)

	resp, err := d.svc.RequestShipment(context.Background(), customerID, balanceShipmentRequest(shipmentID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, shipmentID, resp.ID)
	d.repo.AssertNotCalled(t, "CreateShipment", mock.Anything, mock.Anything)
}

func TestRequestShipment_DuplicateKeyReturnsExistingShipment(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(nil, errcustom.ErrNotFound).Once()
	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateShipment", mock.Anything, mock.Anything).Return(errcustom.ErrShipmentAlreadyExists)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(createdShipment(shipmentID, customerID, quote), nil)

	resp, err := d.svc.RequestShipment(context.Background(), customerID, balanceShipmentRequest(shipmentID, quote.QuoteID))

	require.NoError(t, err)
	require.Equal(t, shipmentID, resp.ID)
	d.matcher.AssertNotCalled(t, "OfferShipment", mock.Anything, mock.Anything)
}

func TestRequestShipment_QuoteNotFoundWithUnusedIDKeepsError(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(nil, errcustom.ErrNotFound)
	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := d.svc.RequestShipment(context.Background(), customerID, balanceShipmentRequest(shipmentID, quote.QuoteID))

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
}

func TestRequestShipment_FailureAfterCreateIsNotMaskedAsRetry(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, shipmentID := uuid.New(), uuid.New()
	quote := sampleShipmentQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	req := balanceShipmentRequest(shipmentID, quote.QuoteID)

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(nil, errcustom.ErrNotFound).Once()
	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateShipment", mock.Anything, mock.Anything).Return(nil)
	d.balance.On("DebitForReference", mock.Anything, customerID, shipmentID, models.ReferenceTypeShipment, expectedBs).Return(errcustom.ErrInsufficientBalance)
	d.repo.On("UpdateShipmentStatus", mock.Anything, shipmentID, models.ShipmentStatusSearching, models.ShipmentStatusCancelled).Return(true, nil)

	_, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
	d.repo.AssertNumberOfCalls(t, "GetShipmentByID", 1)
}
