package e2e_test

import (
	"context"
	"testing"

	paymentsmodels "Encargalo.app-api.go/internal/payments/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFinance_TripCreditDoesNotExposeOrderID(t *testing.T) {
	customerID := newCustomer(t)
	tripID := uuid.New()
	txID, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, tripID, "trip", 80, bcvRate)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, txID)

	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.Len(t, balance.Transactions, 1)
	require.Nil(t, balance.Transactions[0].OrderID)
	require.Equal(t, "payment_overage", balance.Transactions[0].MovementType)

	rows := ledgerForReference(t, tripID)
	require.Len(t, rows, 1)
	require.Equal(t, "trip", *rows[0].ReferenceType)
	require.Nil(t, rows[0].OrderID)

	_, err = reviewsApp.UpsertPagoMovil(context.Background(), customerID, pagoMovil())
	require.NoError(t, err)
	refund, err := reviewsApp.CreateRefundRequest(context.Background(), customerID, txID)
	require.NoError(t, err)
	require.Nil(t, refund.OrderID)

	var refType string
	err = sqlDB.QueryRowContext(context.Background(), `
		SELECT reference_type FROM finance.balance_refund_requests WHERE transaction_id = $1
	`, txID).Scan(&refType)
	require.NoError(t, err)
	require.Equal(t, "trip", refType)
}

func TestFinance_ShopRejectCreditUsesOrderReference(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	totalBs := 400.0

	txID, _, err := reviewsApp.CreditOrderRejected(
		context.Background(),
		customerID,
		seed.Order.ID,
		paymentsmodels.ReferenceTypeOrder,
		totalBs,
		totalBs,
		bcvRate,
	)
	require.NoError(t, err)

	rows := ledgerForReference(t, seed.Order.ID)
	require.Len(t, rows, 1)
	require.Equal(t, "order_rejected_refund", rows[0].MovementType)
	require.InDelta(t, totalBs, rows[0].Amount, 0.001)
	require.Equal(t, "order", *rows[0].ReferenceType)
	require.Nil(t, rows[0].OrderID)

	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.NotNil(t, balance.Transactions[0].OrderID)
	require.Equal(t, seed.Order.ID.String(), *balance.Transactions[0].OrderID)
	require.Equal(t, txID.String(), balance.Transactions[0].ID)
}
