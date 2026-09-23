package app_test

import (
	"testing"
	"time"

	mockports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	"Encargalo.app-api.go/internal/orders/domain/models"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPaymentProofAlertWorker_Publishes(t *testing.T) {
	repo := mockports.NewPaymentReviewsRepo(t)
	orders := mockports.NewOrdersRepo(t)
	stream := mockports.NewRedisStream(t)
	worker := app.NewPaymentProofAlertWorker(repo, orders, stream)

	reviewID := uuid.New()
	orderID := uuid.New()
	now := time.Now()
	review := paymentsModels.PaymentReview{
		ID:               reviewID,
		ReferenceID:      orderID,
		ReferenceType:    paymentsModels.ReferenceTypeOrder,
		ProofURL:         "https://example.com/proof.jpg",
		ExpectedAmountBs: 1680,
		BcvRate:          40,
		Status:           paymentsModels.ReviewPendingReview,
		CreatedAt:        now.Add(-4 * time.Minute),
	}
	order := &models.Order{
		ID:         orderID,
		ShopID:     uuid.New(),
		CustomerID: uuid.New(),
		TotalPrice: 42,
	}

	alertBeforeMatcher := mock.MatchedBy(func(t time.Time) bool {
		return !t.IsZero()
	})
	repo.On("ListPendingForAlert", mock.Anything, paymentsModels.ReferenceTypeOrder, alertBeforeMatcher).
		Return([]paymentsModels.PaymentReview{review}, nil)
	repo.On("MarkAlerted", mock.Anything, reviewID, mock.Anything, alertBeforeMatcher).Return(true, nil)
	orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	stream.On("PublisherEvent", mock.Anything, mock.MatchedBy(func(data any) bool {
		ev, ok := data.(models.DataEventPaymentProofPending)
		return ok && ev.MessageType == "payment_proof_pending" && ev.ReviewID == reviewID.String()
	})).Return(nil)

	ctx := t.Context()
	worker.Start(ctx)
	time.Sleep(200 * time.Millisecond)
	require.True(t, stream.AssertCalled(t, "PublisherEvent", mock.Anything, mock.Anything))
}
