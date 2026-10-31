package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	mockupload "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mocksshipments "Encargalo.app-api.go/internal/mocks/shipments/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/shipmentfare"
	"Encargalo.app-api.go/internal/shipments/app"
	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type shipmentsDeps struct {
	svc     ports.ShipmentsApp
	repo    *mocksshipments.ShipmentsRepo
	matcher *mocksshipments.ShipmentRiderMatcher
	bcv     *mockbcvports.BcvApp
	quotes  *mocksshipments.ShipmentQuotesRepo
	upload  *mockupload.UploadImage
	reviews *mockpayments.PaymentReviewsApp
	hub     *mockpayments.StaffReviewHub
	riders  *mocksshipments.RiderPublicInfoLookup
	balance *mockpayments.CustomerBalanceCreditor
}

// newShipmentsDeps deja sin mockear (nil) customers/riderNotify/events/fareQuotes/contacts:
// los cuatro primeros son no-op si son nil (ver notifyCustomer/publishShipmentEvent/
// notifyRidersCancelled) y contacts cae a los datos crudos del request — mismo
// patrón que newTripsDeps con sus últimos 7 parámetros.
func newShipmentsDeps(t *testing.T) shipmentsDeps {
	t.Helper()
	repo := mocksshipments.NewShipmentsRepo(t)
	matcher := mocksshipments.NewShipmentRiderMatcher(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mocksshipments.NewShipmentQuotesRepo(t)
	reviews := mockpayments.NewPaymentReviewsApp(t)
	upload := mockupload.NewUploadImage(t)
	hub := mockpayments.NewStaffReviewHub(t)
	riders := mocksshipments.NewRiderPublicInfoLookup(t)
	balance := mockpayments.NewCustomerBalanceCreditor(t)
	return shipmentsDeps{
		svc:     app.NewShipmentsApp(repo, matcher, bcv, quotes, upload, reviews, hub, riders, balance, nil, nil, nil, nil, nil),
		repo:    repo,
		matcher: matcher,
		bcv:     bcv,
		quotes:  quotes,
		reviews: reviews,
		upload:  upload,
		hub:     hub,
		riders:  riders,
		balance: balance,
	}
}

func sampleShipmentContacts() (dtos.ShipmentContact, dtos.ShipmentContact) {
	return dtos.ShipmentContact{IsCustomer: true},
		dtos.ShipmentContact{IsCustomer: false, Name: "Juan Pérez", Phone: "+584121234567"}
}

func balanceShipmentRequest(shipmentID, quoteID uuid.UUID) dtos.CreateShipmentRequest {
	sender, recipient := sampleShipmentContacts()
	return dtos.CreateShipmentRequest{
		ID: shipmentID, QuoteID: quoteID, MethodPayment: models.MethodBalance,
		Sender: sender, Recipient: recipient, PackageDescription: "Una torta de cumpleaños",
	}
}

func paidShipmentRequest(quoteID uuid.UUID) dtos.CreateShipmentRequest {
	sender, recipient := sampleShipmentContacts()
	return dtos.CreateShipmentRequest{
		QuoteID: quoteID, MethodPayment: models.MethodPagoMovil, PaymentProofImage: "proof-b64",
		Sender: sender, Recipient: recipient, PackageDescription: "Una torta de cumpleaños",
	}
}

func mockShipmentBcvRate(bcv *mockbcvports.BcvApp, rate float64) {
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

func sampleShipmentQuote(customerID uuid.UUID) *models.ShipmentQuote {
	now := time.Now()
	return &models.ShipmentQuote{
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
		ExpiresAt:        now.Add(models.ShipmentQuoteTTL),
	}
}

func createdShipment(shipmentID, customerID uuid.UUID, quote *models.ShipmentQuote) *models.Shipment {
	fare, rate := quote.FareUSD, quote.BcvRate
	return &models.Shipment{
		ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusSearching,
		Fare: &fare, BcvRate: &rate, MethodPayment: models.MethodBalance,
		PickupCode: "1234", DropoffCode: "5678",
	}
}

func TestEstimateFare_SavesQuoteAndReturnsFare(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockShipmentBcvRate(d.bcv, 40)

	var saved *models.ShipmentQuote
	d.quotes.On("Save", mock.Anything, mock.MatchedBy(func(q *models.ShipmentQuote) bool {
		return q.CustomerID == customerID &&
			q.PickupAddress == req.PickupAddress &&
			q.DropoffAddress == req.DropoffAddress &&
			q.FareUSD == shipmentfare.DevPlaceholderFee
	}), models.ShipmentQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.ShipmentQuote)
	}).Return(nil)

	resp, err := d.svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, resp.QuoteID)
	require.Equal(t, shipmentfare.DevPlaceholderFee, resp.FareUSD)
	require.Equal(t, shipmentfare.DevPlaceholderFee*40, resp.FareBs)
	require.InDelta(t, geo.HaversineKm(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)*1000, resp.DistanceMeters, 0.001)
	require.WithinDuration(t, time.Now().Add(models.ShipmentQuoteTTL), resp.ExpiresAt, 2*time.Second)
	require.NotNil(t, saved)
	require.Equal(t, resp.QuoteID, saved.QuoteID)
	require.Equal(t, 40.0, resp.BcvRate)
}

func TestEstimateFare_FailsWhenBcvRateUnavailable(t *testing.T) {
	d := newShipmentsDeps(t)
	req := sampleFareEstimateRequest(uuid.New())
	d.bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down"))

	_, err := d.svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
	d.quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestEstimateFare_RejectsCoordinatesOutsideVenezuela(t *testing.T) {
	d := newShipmentsDeps(t)
	req := sampleFareEstimateRequest(uuid.New())
	req.PickupLatitude, req.PickupLongitude = 40.7128, -74.0060

	_, err := d.svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
	d.quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestShipment_BalanceDebitsFullFareAndOffers(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID := uuid.New()
	quote := sampleShipmentQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	req := dtos.CreateShipmentRequest{QuoteID: quote.QuoteID, MethodPayment: models.MethodBalance,
		Sender: dtos.ShipmentContact{IsCustomer: true}, Recipient: dtos.ShipmentContact{IsCustomer: true},
		PackageDescription: "Una torta"}

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateShipment", mock.Anything, mock.MatchedBy(func(s *models.Shipment) bool {
		return s.Status == models.ShipmentStatusSearching &&
			s.BalancePaidBs != nil && *s.BalancePaidBs == expectedBs &&
			s.Fare != nil && *s.Fare == quote.FareUSD &&
			s.PickupCode != "" && s.DropoffCode != "" && s.PickupCode != s.DropoffCode
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Shipment).ID = uuid.New()
	}).Return(nil)
	d.balance.On("DebitForReference", mock.Anything, customerID, mock.AnythingOfType("uuid.UUID"), models.ReferenceTypeShipment, expectedBs).Return(nil)
	d.matcher.On("OfferShipment", mock.Anything, mock.MatchedBy(func(s models.Shipment) bool {
		return s.Status == models.ShipmentStatusSearching
	})).Return()

	resp, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, string(models.ShipmentStatusSearching), resp.Status)
	require.Equal(t, expectedBs, *resp.FareBs)
	require.NotEmpty(t, resp.PickupCode)
	require.NotEmpty(t, resp.DropoffCode)
	d.reviews.AssertNotCalled(t, "CreateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestShipment_InsufficientBalanceKeepsQuote(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID := uuid.New()
	quote := sampleShipmentQuote(customerID)
	req := dtos.CreateShipmentRequest{QuoteID: quote.QuoteID, MethodPayment: models.MethodBalance,
		Sender: dtos.ShipmentContact{IsCustomer: true}, Recipient: dtos.ShipmentContact{IsCustomer: true},
		PackageDescription: "Una torta"}

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(1.0, nil)

	_, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateShipment", mock.Anything, mock.Anything)
}

func TestRequestShipment_PagoMovilUsesQuoteFareAndDoesNotOffer(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID := uuid.New()
	quote := sampleShipmentQuote(customerID)
	req := paidShipmentRequest(quote.QuoteID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).
		Return(&uploader.UploadResult{SecureURL: "https://cdn/proof.jpg"}, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateShipment", mock.Anything, mock.MatchedBy(func(s *models.Shipment) bool {
		return s.CustomerID == customerID &&
			s.Status == models.ShipmentStatusPaymentReview &&
			s.Fare != nil && *s.Fare == quote.FareUSD &&
			s.BcvRate != nil && *s.BcvRate == quote.BcvRate
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Shipment).ID = uuid.New()
	}).Return(nil)
	d.reviews.On("CreateReview", mock.Anything, models.ReferenceTypeShipment, mock.AnythingOfType("uuid.UUID"), "https://cdn/proof.jpg", expectedBs, quote.BcvRate).
		Return(&paymentsModels.PaymentReview{
			ID: uuid.New(), ReferenceType: models.ReferenceTypeShipment, ProofURL: "https://cdn/proof.jpg",
			ExpectedAmountBs: expectedBs, Status: paymentsModels.ReviewPendingReview,
		}, nil)
	d.hub.On("BroadcastReview", mock.Anything).Return(nil)
	// A propósito no se mockea bcv.GetRate: si RequestShipment la llamara, haría panic.

	resp, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, string(models.ShipmentStatusPaymentReview), resp.Status)
	require.NotNil(t, resp.FareUSD)
	require.Equal(t, quote.FareUSD, *resp.FareUSD)
	d.matcher.AssertNotCalled(t, "OfferShipment", mock.Anything, mock.Anything)
}

func TestRequestShipment_QuoteNotFoundSkipsProofUpload(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, quoteID := uuid.New(), uuid.New()
	req := paidShipmentRequest(quoteID)

	d.quotes.On("Get", mock.Anything, customerID, quoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateShipment", mock.Anything, mock.Anything)
}

func TestRequestShipment_ReviewFailureCancelsShipment(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID := uuid.New()
	quote := sampleShipmentQuote(customerID)
	req := paidShipmentRequest(quote.QuoteID)
	shipmentID := uuid.New()

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).
		Return(&uploader.UploadResult{SecureURL: "https://cdn/proof.jpg"}, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateShipment", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Shipment).ID = shipmentID
	}).Return(nil)
	d.reviews.On("CreateReview", mock.Anything, models.ReferenceTypeShipment, shipmentID, "https://cdn/proof.jpg", mock.AnythingOfType("float64"), quote.BcvRate).
		Return(nil, errcustom.ErrUnexpectedError)
	d.repo.On("UpdateShipmentStatus", mock.Anything, shipmentID, models.ShipmentStatusPaymentReview, models.ShipmentStatusCancelled).Return(true, nil)

	_, err := d.svc.RequestShipment(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	d.matcher.AssertNotCalled(t, "OfferShipment", mock.Anything, mock.Anything)
}

func TestGetShipment_ForbiddenWhenNotOwner(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, ownerID, requesterID := uuid.New(), uuid.New(), uuid.New()

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{ID: shipmentID, CustomerID: ownerID}, nil)

	_, err := d.svc.GetShipment(context.Background(), shipmentID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrShipmentForbidden)
}

func TestGetShipment_NotFound(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, requesterID := uuid.New(), uuid.New()

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(nil, errcustom.ErrNotFound)

	_, err := d.svc.GetShipment(context.Background(), shipmentID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestGetShipment_OmitsRiderWhenUnassigned(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID := uuid.New(), uuid.New()
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{
		ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusSearching,
		PickupCode: "1234", DropoffCode: "5678",
	}, nil)
	mockShipmentBcvRate(d.bcv, 40)

	resp, err := d.svc.GetShipment(context.Background(), shipmentID, customerID)

	require.NoError(t, err)
	require.Nil(t, resp.Rider)
	require.Equal(t, "1234", resp.PickupCode)
	require.Equal(t, "5678", resp.DropoffCode)
	d.riders.AssertNotCalled(t, "GetByRiderID", mock.Anything, mock.Anything)
}

func TestGetShipment_AttachesPublicRider(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID, riderID := uuid.New(), uuid.New(), uuid.New()
	plate := "AB123CD"
	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{
		ID: shipmentID, CustomerID: customerID, RiderID: &riderID, Status: models.ShipmentStatusAccepted,
		PickupCode: "1234", DropoffCode: "5678",
	}, nil)
	mockShipmentBcvRate(d.bcv, 40)
	d.riders.On("GetByRiderID", mock.Anything, riderID).Return(dtos.RiderPublicInfo{
		Name: "Ada", SurName: "Lovelace", Phone: "04120000000", VehiclePlate: &plate,
	}, nil)

	resp, err := d.svc.GetShipment(context.Background(), shipmentID, customerID)

	require.NoError(t, err)
	require.NotNil(t, resp.Rider)
	require.Equal(t, "Ada", resp.Rider.Name)
	require.Equal(t, &plate, resp.Rider.VehiclePlate)
	// Caso explícito: los códigos salen presentes aunque el envío ya tenga rider y esté accepted.
	require.Equal(t, "1234", resp.PickupCode)
	require.Equal(t, "5678", resp.DropoffCode)
}

func TestGetShipmentHistory_AttachesRiderOnlyWhenAssigned(t *testing.T) {
	d := newShipmentsDeps(t)
	customerID, riderID := uuid.New(), uuid.New()
	assigned := uuid.New()
	d.repo.On("GetShipmentHistoryByCustomer", mock.Anything, customerID).Return([]models.Shipment{
		{ID: uuid.New(), CustomerID: customerID, Status: models.ShipmentStatusDelivered},
		{ID: assigned, CustomerID: customerID, RiderID: &riderID, Status: models.ShipmentStatusDelivered},
	}, nil)
	mockShipmentBcvRate(d.bcv, 40)
	d.riders.On("GetByRiderID", mock.Anything, riderID).Return(dtos.RiderPublicInfo{Name: "Ada"}, nil)

	resp, err := d.svc.GetShipmentHistory(context.Background(), customerID)

	require.NoError(t, err)
	require.Nil(t, resp.Shipments[0].Rider)
	require.Equal(t, "Ada", resp.Shipments[1].Rider.Name)
}

func TestCancelShipment_ForbiddenWhenNotOwner(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, ownerID, requesterID := uuid.New(), uuid.New(), uuid.New()

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{ID: shipmentID, CustomerID: ownerID, Status: models.ShipmentStatusRequested}, nil)

	_, err := d.svc.CancelShipment(context.Background(), shipmentID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrShipmentForbidden)
	d.repo.AssertNotCalled(t, "UpdateShipmentStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelShipment_PaymentReviewPendingRejected(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID := uuid.New(), uuid.New()

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusPaymentReview}, nil)

	_, err := d.svc.CancelShipment(context.Background(), shipmentID, customerID)

	require.ErrorIs(t, err, errcustom.ErrShipmentPaymentReviewPending)
	d.repo.AssertNotCalled(t, "UpdateShipmentStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelShipment_RejectsTerminalStatus(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID := uuid.New(), uuid.New()

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusDelivered}, nil)

	_, err := d.svc.CancelShipment(context.Background(), shipmentID, customerID)

	require.ErrorIs(t, err, errcustom.ErrShipmentNotCancellable)
	d.repo.AssertNotCalled(t, "UpdateShipmentStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelShipment_CASLossReturnsNotCancellable(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID := uuid.New(), uuid.New()

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusRequested}, nil)
	d.repo.On("UpdateShipmentStatus", mock.Anything, shipmentID, models.ShipmentStatusRequested, models.ShipmentStatusCancelled).Return(false, nil)

	_, err := d.svc.CancelShipment(context.Background(), shipmentID, customerID)

	require.ErrorIs(t, err, errcustom.ErrShipmentNotCancellable)
}

func TestCancelShipment_RefundsBalance(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID := uuid.New(), uuid.New()
	balancePaid := 32.0

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{
		ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusSearching,
		MethodPayment: models.MethodBalance, BalancePaidBs: &balancePaid,
	}, nil)
	d.repo.On("UpdateShipmentStatus", mock.Anything, shipmentID, models.ShipmentStatusSearching, models.ShipmentStatusCancelled).Return(true, nil)
	d.repo.On("SaveShipmentCancelReason", mock.Anything, shipmentID, models.CancelReasonCustomerCancelled).Return(nil).Maybe()
	d.balance.On("RefundBalancePayment", mock.Anything, customerID, shipmentID, models.ReferenceTypeShipment, balancePaid).Return(nil)
	mockShipmentBcvRate(d.bcv, 40)

	resp, err := d.svc.CancelShipment(context.Background(), shipmentID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.ShipmentStatusCancelled), resp.Status)
	require.Equal(t, string(models.CancelReasonCustomerCancelled), resp.CancelReason)
}

// TestCancelShipment_CreditsUnservedPagoMovil: un Pago Móvil ya aprobado por
// el staff que se cancela antes de entregarse se acredita a la cartera, igual
// que trips (ver CreditUnservedPagoMovil).
func TestCancelShipment_CreditsUnservedPagoMovil(t *testing.T) {
	d := newShipmentsDeps(t)
	shipmentID, customerID := uuid.New(), uuid.New()
	fare, rate := 0.8, 40.0

	d.repo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&models.Shipment{
		ID: shipmentID, CustomerID: customerID, Status: models.ShipmentStatusSearching,
		MethodPayment: models.MethodPagoMovil, Fare: &fare, BcvRate: &rate,
	}, nil)
	d.repo.On("UpdateShipmentStatus", mock.Anything, shipmentID, models.ShipmentStatusSearching, models.ShipmentStatusCancelled).Return(true, nil)
	d.repo.On("SaveShipmentCancelReason", mock.Anything, shipmentID, models.CancelReasonCustomerCancelled).Return(nil).Maybe()
	d.balance.On("CreditOrderRejected", mock.Anything, customerID, shipmentID, models.ReferenceTypeShipment, money.RoundBs(fare*rate), money.RoundBs(fare*rate), rate).
		Return(uuid.New(), time.Now(), nil)
	d.repo.On("SaveShipmentPayment", mock.Anything, shipmentID, mock.Anything).Return(nil).Maybe()
	mockShipmentBcvRate(d.bcv, rate)

	resp, err := d.svc.CancelShipment(context.Background(), shipmentID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.ShipmentStatusCancelled), resp.Status)
	d.balance.AssertNotCalled(t, "RefundBalancePayment", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
