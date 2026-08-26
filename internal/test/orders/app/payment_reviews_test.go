package app_test

import (
	"context"
	"testing"
	"time"

	mockordersports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type paymentReviewsMocks struct {
	repo    *mockordersports.PaymentReviewsRepo
	orders  *mockordersports.OrdersRepo
	finance *mockordersports.CustomerBalanceCreditor
	stream  *mockordersports.RedisStream
	hub     *mockordersports.StaffReviewHub
	mgr     *mockordersports.AppManagerOrder
}

func newPaymentReviewsFixture(t *testing.T) (ports.PaymentReviewsApp, *paymentReviewsMocks) {
	t.Helper()
	m := &paymentReviewsMocks{
		repo:    mockordersports.NewPaymentReviewsRepo(t),
		orders:  mockordersports.NewOrdersRepo(t),
		finance: mockordersports.NewCustomerBalanceCreditor(t),
		stream:  mockordersports.NewRedisStream(t),
		hub:     mockordersports.NewStaffReviewHub(t),
		mgr:     mockordersports.NewAppManagerOrder(t),
	}
	svc := app.NewPaymentReviewsApp(m.repo, m.orders, m.finance, m.stream, m.hub, m.mgr)
	return svc, m
}

func TestPaymentReviewsApp_CreateForOrder_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	orderID, customerID, shopID := uuid.New(), uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, CustomerID: customerID, ShopID: shopID, TotalPrice: 22}

	m.repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)
	m.hub.On("BroadcastReview", mock.Anything).Return(nil)

	review, err := svc.CreateForOrder(context.Background(), order, "https://cloudinary/proof.jpg", 880, 40)

	require.NoError(t, err)
	assert.Equal(t, ordermodels.ReviewPendingReview, review.Status)
	assert.Equal(t, orderID, review.OrderID)
}

func TestPaymentReviewsApp_Approve_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusPending)).Return(nil)
	m.orders.On("MarkPendingAt", mock.Anything, orderID, mock.Anything).Return(nil)
	m.mgr.On("BroadcastOrder", mock.Anything).Return(nil)

	result, err := svc.Approve(context.Background(), reviewID, staffID, "1234")

	require.NoError(t, err)
	assert.Equal(t, "approve", result.Action)
	assert.Equal(t, string(ordermodels.StatusPending), result.Order.Status)
}

func TestPaymentReviewsApp_Approve_InvalidLast4Format(t *testing.T) {
	svc, _ := newPaymentReviewsFixture(t)

	_, err := svc.Approve(context.Background(), uuid.New(), uuid.New(), "12a4")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_Approve_DuplicateReferenceAutoRejects(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(true, nil)
	// prepareReferenceLast4 delega en RejectInvalid cuando el last4 ya fue usado.
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusRejected)).Return(nil)
	m.orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	m.mgr.On("BroadcastOrderToCustomer", mock.Anything).Return(nil)

	result, err := svc.Approve(context.Background(), reviewID, staffID, "1234")

	require.NoError(t, err)
	assert.True(t, result.DuplicateReject)
	assert.Equal(t, "reject_invalid", result.Action)
}

func TestPaymentReviewsApp_ApproveOverpay_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview, ExpectedAmountBs: 800, BcvRate: 40}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.finance.On("CreditPaymentOverage", mock.Anything, customerID, orderID, 80.0, 40.0).Return(uuid.New(), time.Now(), nil)
	m.orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	m.orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusPending)).Return(nil)
	m.orders.On("MarkPendingAt", mock.Anything, orderID, mock.Anything).Return(nil)
	m.mgr.On("BroadcastOrder", mock.Anything).Return(nil)

	result, err := svc.ApproveOverpay(context.Background(), reviewID, staffID, 880, "1234")

	require.NoError(t, err)
	assert.Equal(t, "approve_overpay", result.Action)
}

func TestPaymentReviewsApp_ApproveOverpay_ObservedBelowExpected(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview, ExpectedAmountBs: 800}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ApproveOverpay(context.Background(), reviewID, staffID, 800, "1234")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_RejectUnderpaid_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview, ExpectedAmountBs: 800, BcvRate: 40}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.finance.On("CreditPaymentPartial", mock.Anything, customerID, orderID, 600.0, 40.0).Return(uuid.New(), time.Now(), nil)
	m.orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusRejected)).Return(nil)
	m.orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	m.mgr.On("BroadcastOrderToCustomer", mock.Anything).Return(nil)

	result, err := svc.RejectUnderpaid(context.Background(), reviewID, staffID, 600, "1234")

	require.NoError(t, err)
	assert.Equal(t, "reject_underpaid", result.Action)
	assert.Equal(t, string(ordermodels.StatusRejected), result.Order.Status)
}

func TestPaymentReviewsApp_RejectUnderpaid_ObservedAboveExpected(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview, ExpectedAmountBs: 800}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.RejectUnderpaid(context.Background(), reviewID, staffID, 800, "1234")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_RejectInvalid_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, staffID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := &ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusPaymentReview)}

	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.orders.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusRejected)).Return(nil)
	m.orders.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	m.mgr.On("BroadcastOrderToCustomer", mock.Anything).Return(nil)

	out, err := svc.RejectInvalid(context.Background(), reviewID, staffID, "comprobante ilegible")

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusRejected), out.Status)
}

func TestPaymentReviewsApp_RejectInvalid_EmptyReason(t *testing.T) {
	svc, _ := newPaymentReviewsFixture(t)

	_, err := svc.RejectInvalid(context.Background(), uuid.New(), uuid.New(), "")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_ListActiveForStaff_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID, shopID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview}
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID}

	m.repo.On("ListActiveForStaff", mock.Anything).Return([]ordermodels.PaymentReview{review}, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	out, err := svc.ListActiveForStaff(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, reviewID, out[0].ReviewID)
}

func TestPaymentReviewsApp_ListActiveForStaff_SkipsOrderLookupFailures(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, orderID := uuid.New(), uuid.New()
	review := ordermodels.PaymentReview{ID: reviewID, OrderID: orderID, Status: ordermodels.ReviewPendingReview}

	m.repo.On("ListActiveForStaff", mock.Anything).Return([]ordermodels.PaymentReview{review}, nil)
	m.orders.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)

	out, err := svc.ListActiveForStaff(context.Background())

	require.NoError(t, err)
	assert.Empty(t, out)
}
