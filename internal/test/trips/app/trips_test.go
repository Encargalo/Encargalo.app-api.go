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
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type tripsDeps struct {
	svc     ports.TripsApp
	repo    *mockstrips.TripsRepo
	matcher *mockstrips.TripRiderMatcher
	bcv     *mockbcvports.BcvApp
	quotes  *mockstrips.TripQuotesRepo
	reviews *mockpayments.PaymentReviewsApp
	upload  *mockupload.UploadImage
	hub     *mockpayments.StaffReviewHub
	riders  *mockstrips.RiderPublicInfoLookup
	balance *mockpayments.CustomerBalanceCreditor
}

func newTripsDeps(t *testing.T) tripsDeps {
	t.Helper()
	repo := mockstrips.NewTripsRepo(t)
	matcher := mockstrips.NewTripRiderMatcher(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	reviews := mockpayments.NewPaymentReviewsApp(t)
	upload := mockupload.NewUploadImage(t)
	hub := mockpayments.NewStaffReviewHub(t)
	riders := mockstrips.NewRiderPublicInfoLookup(t)
	balance := mockpayments.NewCustomerBalanceCreditor(t)
	return tripsDeps{
		svc:     app.NewTripsApp(repo, matcher, bcv, quotes, upload, reviews, hub, riders, balance, nil, nil, nil, nil, nil, nil),
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

func newTripsApp(t *testing.T) (ports.TripsApp, *mockstrips.TripsRepo, *mockstrips.TripRiderMatcher, *mockbcvports.BcvApp, *mockstrips.TripQuotesRepo) {
	t.Helper()
	d := newTripsDeps(t)
	return d.svc, d.repo, d.matcher, d.bcv, d.quotes
}

func paidTripRequest(quoteID uuid.UUID) dtos.CreateTripRequest {
	return dtos.CreateTripRequest{
		QuoteID:           quoteID,
		MethodPayment:     models.MethodPagoMovil,
		PaymentProofImage: "proof-b64",
	}
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
	svc, repo, _, bcv, quotes := newTripsApp(t)
	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.MatchedBy(func(q *models.TripQuote) bool {
		return q.CustomerID == customerID &&
			q.PickupAddress == req.PickupAddress &&
			q.DropoffAddress == req.DropoffAddress &&
			q.FareUSD == tripfare.DevPlaceholderFee
	}), models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, resp.QuoteID)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
	require.Equal(t, tripfare.DevPlaceholderFee*40, resp.FareBs)
	require.InDelta(t, geo.HaversineKm(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)*1000, resp.DistanceMeters, 0.001)
	require.WithinDuration(t, time.Now().Add(models.TripQuoteTTL), resp.ExpiresAt, 2*time.Second)
	require.NotNil(t, saved)
	require.Equal(t, resp.QuoteID, saved.QuoteID)
	require.Equal(t, saved.BcvRate, resp.BcvRate)
	require.Equal(t, 40.0, resp.BcvRate)
}

func TestEstimateFare_FailsWhenBcvRateUnavailable(t *testing.T) {
	svc, _, _, bcv, quotes := newTripsApp(t)
	req := sampleFareEstimateRequest(uuid.New())
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down"))

	_, err := svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
	quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestTrip_UsesQuoteFareAndDoesNotStartMatching(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	req := paidTripRequest(quote.QuoteID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).
		Return(&uploader.UploadResult{SecureURL: "https://cdn/proof.jpg"}, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.CustomerID == customerID &&
			trip.Status == models.TripStatusPaymentReview &&
			trip.PickupAddress == quote.PickupAddress &&
			trip.Fare != nil && *trip.Fare == quote.FareUSD &&
			trip.BcvRate != nil && *trip.BcvRate == quote.BcvRate
	})).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = uuid.New()
	}).Return(nil)
	d.reviews.On("CreateReview", mock.Anything, models.ReferenceTypeTrip, mock.AnythingOfType("uuid.UUID"), "https://cdn/proof.jpg", expectedBs, quote.BcvRate).
		Return(&paymentsModels.PaymentReview{
			ID:               uuid.New(),
			ReferenceType:    models.ReferenceTypeTrip,
			ProofURL:         "https://cdn/proof.jpg",
			ExpectedAmountBs: expectedBs,
			Status:           paymentsModels.ReviewPendingReview,
		}, nil)
	d.hub.On("BroadcastReview", mock.Anything).Return(nil)
	// A propósito no se mockea bcv.GetRate: si RequestTrip la llamara, haría panic.

	resp, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusPaymentReview), resp.Status)
	require.NotNil(t, resp.FareUSD)
	require.Equal(t, quote.FareUSD, *resp.FareUSD)
	require.NotNil(t, resp.FareBs)
	require.Equal(t, expectedBs, *resp.FareBs)
	d.matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_QuoteNotFoundSkipsProofUpload(t *testing.T) {
	d := newTripsDeps(t)
	customerID, quoteID := uuid.New(), uuid.New()
	req := paidTripRequest(quoteID)

	d.quotes.On("Get", mock.Anything, customerID, quoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_UploadFailureDoesNotConsumeQuote(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	req := paidTripRequest(quote.QuoteID)

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).Return(nil, errcustom.ErrUploadingImage)

	_, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrUploadingImage)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
}

// TestRequestTrip_QuoteScopedToRequester: la key de Redis incluye customerID,
// así que un quote_id ajeno a quien llama ni siquiera matchea (no hay forma
// de que el repo devuelva la quote de otro cliente); el repo simplemente no
// la encuentra, igual que si nunca hubiera existido.
func TestRequestTrip_QuoteScopedToRequester(t *testing.T) {
	requesterID, someQuoteID := uuid.New(), uuid.New()

	req := paidTripRequest(someQuoteID)
	d := newTripsDeps(t)
	d.quotes.On("Get", mock.Anything, requesterID, someQuoteID).Return(nil, errcustom.ErrQuoteNotFound)

	_, err := d.svc.RequestTrip(context.Background(), requesterID, req)

	require.ErrorIs(t, err, errcustom.ErrQuoteNotFound)
	d.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_ReviewFailureCancelsTrip(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	req := paidTripRequest(quote.QuoteID)
	tripID := uuid.New()

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).
		Return(&uploader.UploadResult{SecureURL: "https://cdn/proof.jpg"}, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = tripID
	}).Return(nil)
	d.reviews.On("CreateReview", mock.Anything, models.ReferenceTypeTrip, tripID, "https://cdn/proof.jpg", mock.AnythingOfType("float64"), quote.BcvRate).
		Return(nil, errcustom.ErrUnexpectedError)
	d.repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusPaymentReview, models.TripStatusCancelled).Return(true, nil)
	d.repo.On("SaveTripCancelReason", mock.Anything, tripID, mock.Anything).Return(nil).Maybe()

	_, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	d.matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
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

func TestGetTrip_OmitsRiderWhenUnassigned(t *testing.T) {
	d := newTripsDeps(t)
	tripID, customerID := uuid.New(), uuid.New()
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{
		ID: tripID, CustomerID: customerID, Status: models.TripStatusSearching,
	}, nil)
	mockBcvRate(d.bcv, 40)

	resp, err := d.svc.GetTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.Nil(t, resp.Rider)
	d.riders.AssertNotCalled(t, "GetByRiderID", mock.Anything, mock.Anything)
}

func TestGetTrip_AttachesPublicRider(t *testing.T) {
	d := newTripsDeps(t)
	tripID, customerID, riderID := uuid.New(), uuid.New(), uuid.New()
	plate := "AB123CD"
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{
		ID: tripID, CustomerID: customerID, RiderID: &riderID, Status: models.TripStatusAccepted,
	}, nil)
	mockBcvRate(d.bcv, 40)
	d.riders.On("GetByRiderID", mock.Anything, riderID).Return(dtos.RiderPublicInfo{
		Name: "Ada", SurName: "Lovelace", Phone: "04120000000", VehiclePlate: &plate,
	}, nil)

	resp, err := d.svc.GetTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.NotNil(t, resp.Rider)
	require.Equal(t, "Ada", resp.Rider.Name)
	require.Equal(t, &plate, resp.Rider.VehiclePlate)
}

func TestGetTripHistory_AttachesRiderOnlyWhenAssigned(t *testing.T) {
	d := newTripsDeps(t)
	customerID, riderID := uuid.New(), uuid.New()
	assigned := uuid.New()
	d.repo.On("GetTripHistoryByCustomer", mock.Anything, customerID).Return([]models.Trip{
		{ID: uuid.New(), CustomerID: customerID, Status: models.TripStatusCompleted},
		{ID: assigned, CustomerID: customerID, RiderID: &riderID, Status: models.TripStatusCompleted},
	}, nil)
	mockBcvRate(d.bcv, 40)
	d.riders.On("GetByRiderID", mock.Anything, riderID).Return(dtos.RiderPublicInfo{Name: "Ada"}, nil)

	resp, err := d.svc.GetTripHistory(context.Background(), customerID)

	require.NoError(t, err)
	require.Nil(t, resp.Trips[0].Rider)
	require.Equal(t, "Ada", resp.Trips[1].Rider.Name)
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
	repo.On("SaveTripCancelReason", mock.Anything, tripID, mock.Anything).Return(nil).Maybe()
	mockBcvRate(bcv, 40)

	resp, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusCancelled), resp.Status)
}

// TestQuoteFlow_EndToEndFareNeverDrifts: lo que EstimateFare congela es lo que RequestTrip usa después.
func TestQuoteFlow_EndToEndFareNeverDrifts(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	estimateReq := sampleFareEstimateRequest(customerID)
	mockBcvRate(d.bcv, 40)
	d.repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)

	var saved *models.TripQuote
	d.quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	estimate, err := d.svc.EstimateFare(context.Background(), estimateReq)
	require.NoError(t, err)
	require.NotNil(t, saved)

	req := paidTripRequest(estimate.QuoteID)
	d.quotes.On("Get", mock.Anything, customerID, estimate.QuoteID).Return(saved, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).
		Return(&uploader.UploadResult{SecureURL: "https://cdn/proof.jpg"}, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, estimate.QuoteID).Return(saved, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.Status == models.TripStatusPaymentReview && trip.Fare != nil && *trip.Fare == estimate.FareUSD
	})).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = uuid.New()
	}).Return(nil)
	d.reviews.On("CreateReview", mock.Anything, models.ReferenceTypeTrip, mock.AnythingOfType("uuid.UUID"), "https://cdn/proof.jpg", estimate.FareBs, saved.BcvRate).
		Return(&paymentsModels.PaymentReview{ID: uuid.New(), ExpectedAmountBs: estimate.FareBs, ProofURL: "https://cdn/proof.jpg"}, nil)
	d.hub.On("BroadcastReview", mock.Anything).Return(nil)

	resp, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusPaymentReview), resp.Status)
	require.NotNil(t, resp.FareUSD)
	require.Equal(t, estimate.FareUSD, *resp.FareUSD)
	require.NotNil(t, resp.FareBs)
	require.Equal(t, estimate.FareBs, *resp.FareBs)
	d.matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
}

func TestRequestTrip_BalanceDebitsFullFareAndOffers(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	req := dtos.CreateTripRequest{QuoteID: quote.QuoteID, MethodPayment: models.MethodBalance}

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(expectedBs, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.Status == models.TripStatusSearching &&
			trip.BalancePaidBs != nil && *trip.BalancePaidBs == expectedBs &&
			trip.Fare != nil && *trip.Fare == quote.FareUSD
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Trip).ID = uuid.New()
	}).Return(nil)
	d.balance.On("DebitForReference", mock.Anything, customerID, mock.AnythingOfType("uuid.UUID"), models.ReferenceTypeTrip, expectedBs).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.MatchedBy(func(trip models.Trip) bool {
		return trip.Status == models.TripStatusSearching
	})).Return()

	resp, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusSearching), resp.Status)
	require.Equal(t, expectedBs, *resp.FareBs)
	d.reviews.AssertNotCalled(t, "CreateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestTrip_InsufficientBalanceKeepsQuote(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	req := dtos.CreateTripRequest{QuoteID: quote.QuoteID, MethodPayment: models.MethodBalance}

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.balance.On("GetBalanceBs", mock.Anything, customerID).Return(1.0, nil)

	_, err := d.svc.RequestTrip(context.Background(), customerID, req)

	require.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
	d.quotes.AssertNotCalled(t, "GetAndConsume", mock.Anything, mock.Anything, mock.Anything)
	d.repo.AssertNotCalled(t, "CreateTrip", mock.Anything, mock.Anything)
}
