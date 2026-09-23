package app_test

import (
	"context"
	"testing"
	"time"

	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	"Encargalo.app-api.go/internal/payments/app"
	"Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type paymentReviewsFixture struct {
	repo    *mockpayments.PaymentReviewsRepo
	finance *mockpayments.CustomerBalanceCreditor
	hub     *mockpayments.StaffReviewHub
	subject *mockpayments.PaymentSubject
}

func newPaymentReviewsFixture(t *testing.T) (ports.PaymentReviewsApp, *paymentReviewsFixture) {
	t.Helper()
	m := &paymentReviewsFixture{
		repo:    mockpayments.NewPaymentReviewsRepo(t),
		finance: mockpayments.NewCustomerBalanceCreditor(t),
		hub:     mockpayments.NewStaffReviewHub(t),
		subject: mockpayments.NewPaymentSubject(t),
	}
	registry := app.NewSubjectRegistry()
	registry.Register(models.ReferenceTypeOrder, m.subject)
	svc := app.NewPaymentReviewsApp(m.repo, m.finance, m.hub, registry)
	return svc, m
}

func pendingReview(reviewID, referenceID uuid.UUID) *models.PaymentReview {
	return &models.PaymentReview{
		ID:               reviewID,
		ReferenceID:      referenceID,
		ReferenceType:    models.ReferenceTypeOrder,
		Status:           models.ReviewPendingReview,
		ExpectedAmountBs: 800,
		BcvRate:          40,
	}
}

func TestPaymentReviewsApp_Approve_Success(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, referenceID, staffID := uuid.New(), uuid.New(), uuid.New()
	review := pendingReview(reviewID, referenceID)

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.subject.On("OnApproved", mock.Anything, referenceID, reviewID).Return(nil)
	m.hub.On("BroadcastReview", mock.Anything).Return(nil)

	result, err := svc.Approve(context.Background(), reviewID, staffID, "1234")

	require.NoError(t, err)
	assert.Equal(t, "approve", result.Action)
	assert.Equal(t, referenceID, result.ReferenceID)
	assert.Equal(t, models.ReferenceTypeOrder, result.ReferenceType)
}

func TestPaymentReviewsApp_Approve_InvalidLast4Format(t *testing.T) {
	svc, _ := newPaymentReviewsFixture(t)

	_, err := svc.Approve(context.Background(), uuid.New(), uuid.New(), "12a4")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_Approve_UnknownReferenceType(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID := uuid.New()
	review := pendingReview(reviewID, uuid.New())
	review.ReferenceType = "trip"

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)

	_, err := svc.Approve(context.Background(), reviewID, uuid.New(), "1234")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_Approve_DuplicateReferenceAutoRejects(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, referenceID, staffID := uuid.New(), uuid.New(), uuid.New()
	review := pendingReview(reviewID, referenceID)

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(true, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.subject.On("OnRejected", mock.Anything, referenceID, reviewID, mock.Anything, mock.Anything).Return(nil)
	m.hub.On("BroadcastReview", mock.Anything).Return(nil)

	result, err := svc.Approve(context.Background(), reviewID, staffID, "1234")

	require.NoError(t, err)
	assert.True(t, result.DuplicateReject)
	assert.Equal(t, "reject_invalid", result.Action)
	assert.Contains(t, result.RejectionReason, "1234")
}

func TestPaymentReviewsApp_ApproveOverpay_CreditsWithReferenceType(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, referenceID, staffID, customerID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	review := pendingReview(reviewID, referenceID)

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.subject.On("CustomerID", mock.Anything, referenceID).Return(customerID, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.finance.On("CreditPaymentOverage", mock.Anything, customerID, referenceID, models.ReferenceTypeOrder, 80.0, 40.0).
		Return(uuid.New(), time.Now(), nil)
	m.subject.On("OnOverpaid", mock.Anything, referenceID, reviewID, 80.0, mock.Anything).Return(nil)
	m.hub.On("BroadcastReview", mock.Anything).Return(nil)

	result, err := svc.ApproveOverpay(context.Background(), reviewID, staffID, 880, "1234")

	require.NoError(t, err)
	assert.Equal(t, "approve_overpay", result.Action)
}

func TestPaymentReviewsApp_ApproveOverpay_ObservedBelowExpected(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, referenceID := uuid.New(), uuid.New()
	review := pendingReview(reviewID, referenceID)

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)

	_, err := svc.ApproveOverpay(context.Background(), reviewID, uuid.New(), 800, "1234")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_RejectUnderpaid_CreditsPartial(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID, referenceID, customerID := uuid.New(), uuid.New(), uuid.New()
	review := pendingReview(reviewID, referenceID)

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)
	m.subject.On("CustomerID", mock.Anything, referenceID).Return(customerID, nil)
	m.repo.On("ResolvePending", mock.Anything, reviewID, mock.Anything).Return(true, nil)
	m.finance.On("CreditPaymentPartial", mock.Anything, customerID, referenceID, models.ReferenceTypeOrder, 600.0, 40.0).
		Return(uuid.New(), time.Now(), nil)
	m.subject.On("OnRejected", mock.Anything, referenceID, reviewID, "underpaid", mock.Anything).Return(nil)
	m.hub.On("BroadcastReview", mock.Anything).Return(nil)

	result, err := svc.RejectUnderpaid(context.Background(), reviewID, uuid.New(), 600, "1234")

	require.NoError(t, err)
	assert.Equal(t, "reject_underpaid", result.Action)
}

func TestPaymentReviewsApp_RejectUnderpaid_ObservedAboveExpected(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	reviewID := uuid.New()
	review := pendingReview(reviewID, uuid.New())

	m.repo.On("ExistsResolvedReferenceLast4", mock.Anything, "1234").Return(false, nil)
	m.repo.On("GetByID", mock.Anything, reviewID).Return(review, nil)

	_, err := svc.RejectUnderpaid(context.Background(), reviewID, uuid.New(), 800, "1234")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_RejectInvalid_EmptyReason(t *testing.T) {
	svc, _ := newPaymentReviewsFixture(t)

	_, err := svc.RejectInvalid(context.Background(), uuid.New(), uuid.New(), "")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestPaymentReviewsApp_ListActiveForStaff_UsesSnapshotProvider(t *testing.T) {
	svc, _ := newPaymentReviewsFixture(t)
	reviewID := uuid.New()
	svc.SetSnapshotProvider(func(context.Context) ([]models.PaymentReviewBroadcast, error) {
		return []models.PaymentReviewBroadcast{{ReviewID: reviewID, ReferenceType: models.ReferenceTypeOrder}}, nil
	})

	out, err := svc.ListActiveForStaff(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, reviewID, out[0].ReviewID)
}

func TestPaymentReviewsApp_ListActiveForStaff_GenericFallback(t *testing.T) {
	svc, m := newPaymentReviewsFixture(t)
	review := *pendingReview(uuid.New(), uuid.New())
	m.repo.On("ListActiveForStaff", mock.Anything).Return([]models.PaymentReview{review}, nil)

	out, err := svc.ListActiveForStaff(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, review.ReferenceID, out[0].ReferenceID)
	assert.Equal(t, models.ReferenceTypeOrder, out[0].ReferenceType)
}
