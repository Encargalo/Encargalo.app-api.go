package app_test

import (
	"context"
	"testing"

	mockordersports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStaffReviewHub_RegisterWithoutReviewsApp(t *testing.T) {
	hub := app.NewStaffReviewHub()
	conn := mockordersports.NewManagerConn(t)
	staffID := uuid.New()

	err := hub.Register(context.Background(), staffID, conn)

	require.NoError(t, err)
}

func TestStaffReviewHub_RegisterSendsActiveReviewsSnapshot(t *testing.T) {
	hub := app.NewStaffReviewHub()
	reviewsApp := mockordersports.NewPaymentReviewsApp(t)
	hub.SetReviewsApp(reviewsApp)

	staffID := uuid.New()
	conn := mockordersports.NewManagerConn(t)
	reviewsApp.On("ListActiveForStaff", mock.Anything).Return([]ordermodels.PaymentReviewBroadcast{
		{Type: "payment_review_update", ReviewID: uuid.New()},
	}, nil)
	conn.On("Send", mock.Anything).Return()

	err := hub.Register(context.Background(), staffID, conn)

	require.NoError(t, err)
}

func TestStaffReviewHub_RegisterPropagatesListError(t *testing.T) {
	hub := app.NewStaffReviewHub()
	reviewsApp := mockordersports.NewPaymentReviewsApp(t)
	hub.SetReviewsApp(reviewsApp)

	staffID := uuid.New()
	conn := mockordersports.NewManagerConn(t)
	reviewsApp.On("ListActiveForStaff", mock.Anything).Return(nil, assert.AnError)

	err := hub.Register(context.Background(), staffID, conn)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestStaffReviewHub_UnregisterRemovesConnection(t *testing.T) {
	hub := app.NewStaffReviewHub()
	staffID := uuid.New()
	conn := mockordersports.NewManagerConn(t)

	err := hub.Register(context.Background(), staffID, conn)
	require.NoError(t, err)

	conn.On("Close").Return()
	hub.Unregister(staffID, conn)

	// Tras Unregister no debe quedar conexión: si BroadcastReview la
	// alcanzara, Send haría panic por no tener expectativa configurada.
	err = hub.BroadcastReview(ordermodels.PaymentReviewBroadcast{ReviewID: uuid.New()})
	require.NoError(t, err)
}

func TestStaffReviewHub_UnregisterUnknownStaffClosesConn(t *testing.T) {
	hub := app.NewStaffReviewHub()
	conn := mockordersports.NewManagerConn(t)
	conn.On("Close").Return()

	hub.Unregister(uuid.New(), conn)
}

func TestStaffReviewHub_BroadcastReview_SendsToAllConnections(t *testing.T) {
	hub := app.NewStaffReviewHub()
	staffID := uuid.New()
	conn := mockordersports.NewManagerConn(t)
	require.NoError(t, hub.Register(context.Background(), staffID, conn))

	conn.On("Send", mock.Anything).Return()

	err := hub.BroadcastReview(ordermodels.PaymentReviewBroadcast{Type: "payment_review_update", ReviewID: uuid.New()})

	require.NoError(t, err)
}
