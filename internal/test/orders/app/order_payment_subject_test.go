package app_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mockordersports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOrderPaymentSubject_OnApproved_ReleasesToShop(t *testing.T) {
	orders := mockordersports.NewOrdersRepo(t)
	reviews := mockordersports.NewPaymentReviewsRepo(t)
	mgr := mockordersports.NewAppManagerOrder(t)
	subject := app.NewOrderPaymentSubject(orders, reviews, nil, mgr)

	orderID, reviewID := uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, Status: string(ordermodels.StatusPaymentReview)}
	review := &paymentsModels.PaymentReview{ID: reviewID, ReferenceID: orderID, Status: paymentsModels.ReviewApproved}

	orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	reviews.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusPending)).Return(nil)
	orders.On("MarkPendingAt", mock.Anything, orderID, mock.Anything).Return(nil)
	mgr.On("BroadcastOrder", mock.Anything).Return(nil)

	err := subject.OnApproved(context.Background(), orderID, reviewID)

	require.NoError(t, err)
}

func TestOrderPaymentSubject_OnApproved_RejectsWhenNotInPaymentReview(t *testing.T) {
	orders := mockordersports.NewOrdersRepo(t)
	subject := app.NewOrderPaymentSubject(orders, nil, nil, nil)

	orderID := uuid.New()
	orders.On("SearchOrdersByID", mock.Anything, orderID).Return(&ordermodels.Order{
		ID: orderID, Status: string(ordermodels.StatusPending),
	}, nil)

	err := subject.OnApproved(context.Background(), orderID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestOrderPaymentSubject_OnRejected_RefundsBalanceAndRejects(t *testing.T) {
	orders := mockordersports.NewOrdersRepo(t)
	reviews := mockordersports.NewPaymentReviewsRepo(t)
	finance := mockordersports.NewCustomerBalanceCreditor(t)
	mgr := mockordersports.NewAppManagerOrder(t)
	subject := app.NewOrderPaymentSubject(orders, reviews, finance, mgr)

	orderID, reviewID, customerID := uuid.New(), uuid.New(), uuid.New()
	details, err := json.Marshal(ordermodels.Details{PaymentDetails: &ordermodels.PaymentDetails{BalanceAmountBs: 40}})
	require.NoError(t, err)
	order := &ordermodels.Order{
		ID: orderID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview), Details: details,
	}
	pm := 600.0
	review := &paymentsModels.PaymentReview{ID: reviewID, ReferenceID: orderID, Status: paymentsModels.ReviewRejectedUnderpaid, CreditedBs: &pm, BcvRate: 40}
	credit := paymentsPorts.BalanceCredit{TxID: uuid.New(), RefundablePMBs: 600, CreatedAt: time.Now()}

	orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	reviews.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	finance.On("RefundBalancePayment", mock.Anything, customerID, orderID, paymentsModels.ReferenceTypeOrder, 40.0).Return(nil)
	reviews.On("UpdateCreditedBs", mock.Anything, reviewID, 640.0).Return(nil)
	orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusRejected)).Return(nil)
	mgr.On("BroadcastOrderToCustomer", mock.Anything).Return(nil)

	err = subject.OnRejected(context.Background(), orderID, reviewID, "underpaid", credit)

	require.NoError(t, err)
}

func TestOrderPaymentSubject_OnOverpaid_ReleasesAfterSync(t *testing.T) {
	orders := mockordersports.NewOrdersRepo(t)
	reviews := mockordersports.NewPaymentReviewsRepo(t)
	mgr := mockordersports.NewAppManagerOrder(t)
	subject := app.NewOrderPaymentSubject(orders, reviews, nil, mgr)

	orderID, reviewID := uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, Status: string(ordermodels.StatusPaymentReview)}
	review := &paymentsModels.PaymentReview{ID: reviewID, ReferenceID: orderID, Status: paymentsModels.ReviewOverpaid, BcvRate: 40}
	credit := paymentsPorts.BalanceCredit{TxID: uuid.New(), RefundablePMBs: 80, CreatedAt: time.Now()}

	orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	reviews.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusPending)).Return(nil)
	orders.On("MarkPendingAt", mock.Anything, orderID, mock.Anything).Return(nil)
	mgr.On("BroadcastOrder", mock.Anything).Return(nil)

	err := subject.OnOverpaid(context.Background(), orderID, reviewID, 80, credit)

	require.NoError(t, err)
}
