package app_test

import (
	"context"
	"testing"

	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func paymentReviewTrip(id uuid.UUID) *models.Trip {
	fare := 2.5
	rate := 40.0
	return &models.Trip{
		ID:         id,
		CustomerID: uuid.New(),
		Status:     models.TripStatusPaymentReview,
		Fare:       &fare,
		BcvRate:    &rate,
	}
}

func TestTripPaymentSubject_OnApproved_StartsMatching(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	matcher := mockstrips.NewTripRiderMatcher(t)
	subject := app.NewTripPaymentSubject(repo, matcher, nil)
	tripID := uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(paymentReviewTrip(tripID), nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusPaymentReview, models.TripStatusSearching).Return(true, nil)
	matcher.On("OfferTrip", mock.Anything, mock.MatchedBy(func(trip models.Trip) bool {
		return trip.ID == tripID && trip.Status == models.TripStatusSearching
	})).Return()

	err := subject.OnApproved(context.Background(), tripID, uuid.New())

	require.NoError(t, err)
}

func TestTripPaymentSubject_OnApproved_RejectsWhenNotInPaymentReview(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	matcher := mockstrips.NewTripRiderMatcher(t)
	subject := app.NewTripPaymentSubject(repo, matcher, nil)
	tripID := uuid.New()
	trip := paymentReviewTrip(tripID)
	trip.Status = models.TripStatusSearching

	repo.On("GetTripByID", mock.Anything, tripID).Return(trip, nil)

	err := subject.OnApproved(context.Background(), tripID, uuid.New())

	require.ErrorIs(t, err, errcustom.ErrInvalidStatus)
	matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
}

func TestTripPaymentSubject_OnOverpaid_StartsMatching(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	matcher := mockstrips.NewTripRiderMatcher(t)
	subject := app.NewTripPaymentSubject(repo, matcher, nil)
	tripID := uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(paymentReviewTrip(tripID), nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusPaymentReview, models.TripStatusSearching).Return(true, nil)
	matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	err := subject.OnOverpaid(context.Background(), tripID, uuid.New(), 10, paymentsPorts.BalanceCredit{})

	require.NoError(t, err)
}

func TestTripPaymentSubject_OnRejected_CancelsWithoutMatching(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	matcher := mockstrips.NewTripRiderMatcher(t)
	subject := app.NewTripPaymentSubject(repo, matcher, nil)
	tripID := uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(paymentReviewTrip(tripID), nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusPaymentReview, models.TripStatusCancelled).Return(true, nil)

	err := subject.OnRejected(context.Background(), tripID, uuid.New(), "underpaid", paymentsPorts.BalanceCredit{})

	require.NoError(t, err)
	matcher.AssertNotCalled(t, "OfferTrip", mock.Anything, mock.Anything)
}

func TestTripPaymentSubject_ExpectedAmountBs_UsesFrozenRate(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	subject := app.NewTripPaymentSubject(repo, nil, nil)
	tripID := uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(paymentReviewTrip(tripID), nil)

	amount, err := subject.ExpectedAmountBs(context.Background(), tripID)

	require.NoError(t, err)
	require.Equal(t, 100.0, amount)
}
