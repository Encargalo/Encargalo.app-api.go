package app_test

import (
	"context"
	"testing"

	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	"Encargalo.app-api.go/internal/payments/app"
	"Encargalo.app-api.go/internal/payments/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStaffReviewHub_RegisterWithoutProvider(t *testing.T) {
	hub := app.NewStaffReviewHub(nil)
	conn := mockpayments.NewManagerConn(t)

	err := hub.Register(context.Background(), uuid.New(), conn)

	require.NoError(t, err)
}

func TestStaffReviewHub_RegisterSendsSnapshot(t *testing.T) {
	hub := app.NewStaffReviewHub(nil)
	hub.SetSnapshotProvider(func(context.Context) ([]models.PaymentReviewBroadcast, error) {
		return []models.PaymentReviewBroadcast{{Type: "payment_review_update", ReviewID: uuid.New()}}, nil
	})
	conn := mockpayments.NewManagerConn(t)
	conn.On("Send", mock.Anything).Return()

	err := hub.Register(context.Background(), uuid.New(), conn)

	require.NoError(t, err)
}

func TestStaffReviewHub_RegisterPropagatesSnapshotError(t *testing.T) {
	hub := app.NewStaffReviewHub(nil)
	hub.SetSnapshotProvider(func(context.Context) ([]models.PaymentReviewBroadcast, error) {
		return nil, assert.AnError
	})
	conn := mockpayments.NewManagerConn(t)

	err := hub.Register(context.Background(), uuid.New(), conn)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestStaffReviewHub_UnregisterRemovesConnection(t *testing.T) {
	hub := app.NewStaffReviewHub(nil)
	staffID := uuid.New()
	conn := mockpayments.NewManagerConn(t)

	err := hub.Register(context.Background(), staffID, conn)
	require.NoError(t, err)

	conn.On("Close").Return()
	hub.Unregister(staffID, conn)

	err = hub.BroadcastReview(models.PaymentReviewBroadcast{ReviewID: uuid.New()})
	require.NoError(t, err)
}

func TestStaffReviewHub_UnregisterUnknownStaffClosesConn(t *testing.T) {
	hub := app.NewStaffReviewHub(nil)
	conn := mockpayments.NewManagerConn(t)
	conn.On("Close").Return()

	hub.Unregister(uuid.New(), conn)
}

func TestStaffReviewHub_BroadcastReview_SendsToAllConnections(t *testing.T) {
	hub := app.NewStaffReviewHub(nil)
	staffID := uuid.New()
	conn := mockpayments.NewManagerConn(t)
	require.NoError(t, hub.Register(context.Background(), staffID, conn))

	conn.On("Send", mock.Anything).Return()

	err := hub.BroadcastReview(models.PaymentReviewBroadcast{Type: "payment_review_update", ReviewID: uuid.New()})

	require.NoError(t, err)
}
