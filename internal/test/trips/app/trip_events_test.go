package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeTripEvents struct {
	mu     sync.Mutex
	events []any
}

func (f *fakeTripEvents) PublisherEvent(_ context.Context, data any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, data)
	return nil
}

func TestRequestTrip_CashPublishesTripCreated(t *testing.T) {
	d := newTripsDeps(t)
	events := &fakeTripEvents{}
	svc := app.NewTripsApp(d.repo, d.matcher, d.bcv, d.quotes, d.upload, d.reviews, d.hub, d.riders, d.balance, nil, nil, nil, events, nil, nil)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	tripID := uuid.New()

	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Trip).ID = tripID
	}).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	_, err := svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{
		QuoteID:       quote.QuoteID,
		MethodPayment: models.MethodEfectivo,
	})

	require.NoError(t, err)
	require.Equal(t, []any{models.DataEventTrip{
		MessageType:   models.EventTripCreated,
		TripID:        tripID.String(),
		CustomerID:    customerID.String(),
		FareUSD:       quote.FareUSD,
		MethodPayment: models.MethodEfectivo,
	}}, events.events)
}

func TestRequestTrip_PagoMovilPublishesProofPendingAndTripCreated(t *testing.T) {
	d := newTripsDeps(t)
	events := &fakeTripEvents{}
	svc := app.NewTripsApp(d.repo, d.matcher, d.bcv, d.quotes, d.upload, d.reviews, d.hub, d.riders, d.balance, nil, nil, nil, events, nil, nil)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	req := paidTripRequest(quote.QuoteID)
	expectedBs := money.RoundBs(quote.FareUSD * quote.BcvRate)
	tripID, reviewID := uuid.New(), uuid.New()

	d.quotes.On("Get", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.upload.On("UploadImage", mock.Anything, req.PaymentProofImage).
		Return(&uploader.UploadResult{SecureURL: "https://cdn/proof.jpg"}, nil)
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Trip).ID = tripID
	}).Return(nil)
	d.reviews.On("CreateReview", mock.Anything, models.ReferenceTypeTrip, tripID, "https://cdn/proof.jpg", expectedBs, quote.BcvRate).
		Return(&paymentsModels.PaymentReview{ID: reviewID, ProofURL: "https://cdn/proof.jpg", ExpectedAmountBs: expectedBs}, nil)
	d.hub.On("BroadcastReview", mock.Anything).Return(nil)

	_, err := svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Len(t, events.events, 2)
	require.Equal(t, models.DataEventTripPaymentProofPending{
		MessageType:      "payment_proof_pending",
		ReviewID:         reviewID.String(),
		ReferenceType:    models.ReferenceTypeTrip,
		TripID:           tripID.String(),
		ExpectedAmountBs: expectedBs,
		ProofURL:         "https://cdn/proof.jpg",
	}, events.events[0])
	created, ok := events.events[1].(models.DataEventTrip)
	require.True(t, ok)
	require.Equal(t, models.EventTripCreated, created.MessageType)
	require.Equal(t, models.MethodPagoMovil, created.MethodPayment)
}

func TestTripPaymentProofAlertWorker_RepublishesPendingTripProofs(t *testing.T) {
	reviews := mockpayments.NewPaymentReviewsRepo(t)
	events := &fakeTripEvents{}
	worker := app.NewTripPaymentProofAlertWorker(reviews, events)
	review := paymentsModels.PaymentReview{
		ID:               uuid.New(),
		ReferenceID:      uuid.New(),
		ReferenceType:    models.ReferenceTypeTrip,
		ProofURL:         "https://cdn/proof.jpg",
		ExpectedAmountBs: 100,
		CreatedAt:        time.Now().Add(-6 * time.Minute),
	}
	reviews.On("ListPendingForAlert", mock.Anything, models.ReferenceTypeTrip, mock.Anything).Return([]paymentsModels.PaymentReview{review}, nil)
	reviews.On("MarkAlerted", mock.Anything, review.ID, mock.Anything, mock.Anything).Return(true, nil)

	worker.Tick(context.Background())

	require.Len(t, events.events, 1)
	event := events.events[0].(models.DataEventTripPaymentProofPending)
	require.Equal(t, review.ID.String(), event.ReviewID)
	require.Equal(t, review.ReferenceID.String(), event.TripID)
	require.Equal(t, models.ReferenceTypeTrip, event.ReferenceType)
	require.Equal(t, 6, event.MinutesPending)
}

func TestTripPaymentProofAlertWorker_SkipsAlreadyAlerted(t *testing.T) {
	reviews := mockpayments.NewPaymentReviewsRepo(t)
	events := &fakeTripEvents{}
	worker := app.NewTripPaymentProofAlertWorker(reviews, events)
	review := paymentsModels.PaymentReview{ID: uuid.New(), ReferenceID: uuid.New(), CreatedAt: time.Now()}
	reviews.On("ListPendingForAlert", mock.Anything, models.ReferenceTypeTrip, mock.Anything).Return([]paymentsModels.PaymentReview{review}, nil)
	reviews.On("MarkAlerted", mock.Anything, review.ID, mock.Anything, mock.Anything).Return(false, nil)

	worker.Tick(context.Background())

	require.Empty(t, events.events)
}
