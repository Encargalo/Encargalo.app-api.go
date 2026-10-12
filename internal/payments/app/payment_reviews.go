package app

import (
	"Encargalo.app-api.go/internal/pkg/id"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

var referenceLast4Pattern = regexp.MustCompile(`^[0-9]{4}$`)

type paymentReviewsApp struct {
	repo     ports.PaymentReviewsRepo
	finance  ports.CustomerBalanceCreditor
	hub      ports.StaffReviewHub
	registry ports.PaymentSubjectRegistry
	snapshot func(ctx context.Context) ([]models.PaymentReviewBroadcast, error)
}

func NewPaymentReviewsApp(
	repo ports.PaymentReviewsRepo,
	finance ports.CustomerBalanceCreditor,
	hub ports.StaffReviewHub,
	registry ports.PaymentSubjectRegistry,
) ports.PaymentReviewsApp {
	return &paymentReviewsApp{repo: repo, finance: finance, hub: hub, registry: registry}
}

func (a *paymentReviewsApp) SetSnapshotProvider(fn func(ctx context.Context) ([]models.PaymentReviewBroadcast, error)) {
	a.snapshot = fn
}

// CreateReview persists a pending review. Order creation does not call this:
// it writes the row inside its own rollback path and broadcasts itself.
func (a *paymentReviewsApp) CreateReview(ctx context.Context, referenceType string, referenceID uuid.UUID, proofURL string, expectedBs, bcvRate float64) (*models.PaymentReview, error) {
	if strings.TrimSpace(referenceType) == "" || referenceID == uuid.Nil || strings.TrimSpace(proofURL) == "" || expectedBs <= 0 {
		return nil, errcustom.ErrInvalidStatus
	}
	now := time.Now()
	review := &models.PaymentReview{
		ID:               id.New(),
		ReferenceID:      referenceID,
		ReferenceType:    referenceType,
		ProofURL:         proofURL,
		ExpectedAmountBs: expectedBs,
		BcvRate:          bcvRate,
		Status:           models.ReviewPendingReview,
		CreatedAt:        now,
		LastAlertedAt:    &now,
	}
	if err := a.repo.Create(ctx, review); err != nil {
		return nil, err
	}
	return review, nil
}

func (a *paymentReviewsApp) Approve(ctx context.Context, reviewID, staffID uuid.UUID, referenceLast4 string) (models.StaffReviewResult, error) {
	last4, dup, err := a.prepareReferenceLast4(ctx, reviewID, staffID, referenceLast4)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if dup != nil {
		return *dup, nil
	}

	review, subject, err := a.loadPendingSubject(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}

	update := &models.PaymentReview{
		Status:         models.ReviewApproved,
		StaffID:        &staffID,
		ReferenceLast4: &last4,
	}
	if err := a.resolve(ctx, review, update); err != nil {
		return models.StaffReviewResult{}, err
	}
	if err := subject.OnApproved(ctx, review.ReferenceID, review.ID); err != nil {
		return models.StaffReviewResult{}, err
	}
	a.broadcastResolved(ctx, review, subject)
	return resultOf(review, "approve"), nil
}

func (a *paymentReviewsApp) ApproveOverpay(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error) {
	last4, dup, err := a.prepareReferenceLast4(ctx, reviewID, staffID, referenceLast4)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if dup != nil {
		return *dup, nil
	}

	review, subject, err := a.loadPendingSubject(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if observedBs <= review.ExpectedAmountBs {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}
	customerID, err := subject.CustomerID(ctx, review.ReferenceID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}

	diff := observedBs - review.ExpectedAmountBs
	creditedBs := money.RoundBs(diff)
	update := &models.PaymentReview{
		Status:           models.ReviewOverpaid,
		StaffID:          &staffID,
		ObservedAmountBs: &observedBs,
		DifferenceBs:     &diff,
		CreditedBs:       &creditedBs,
		ReferenceLast4:   &last4,
	}
	if err := a.resolve(ctx, review, update); err != nil {
		return models.StaffReviewResult{}, err
	}

	txID, createdAt, err := a.finance.CreditPaymentOverage(ctx, customerID, review.ReferenceID, review.ReferenceType, creditedBs, review.BcvRate)
	if err != nil {
		slog.Error("ApproveOverpay credit failed", "referenceID", review.ReferenceID, "referenceType", review.ReferenceType, "error", err)
		return models.StaffReviewResult{}, err
	}
	credit := ports.BalanceCredit{TxID: txID, RefundablePMBs: creditedBs, CreatedAt: createdAt}
	if err := subject.OnOverpaid(ctx, review.ReferenceID, review.ID, diff, credit); err != nil {
		return models.StaffReviewResult{}, err
	}
	a.broadcastResolved(ctx, review, subject)
	return resultOf(review, "approve_overpay"), nil
}

// RejectUnderpaid credits the amount actually paid (observedBs) and rejects
// the reference immediately. The customer retries by creating a new order or trip.
func (a *paymentReviewsApp) RejectUnderpaid(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error) {
	last4, dup, err := a.prepareReferenceLast4(ctx, reviewID, staffID, referenceLast4)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if dup != nil {
		return *dup, nil
	}

	review, subject, err := a.loadPendingSubject(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if observedBs >= review.ExpectedAmountBs {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}
	customerID, err := subject.CustomerID(ctx, review.ReferenceID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}

	diff := review.ExpectedAmountBs - observedBs
	pmCreditedBs := money.RoundBs(observedBs)
	update := &models.PaymentReview{
		Status:           models.ReviewRejectedUnderpaid,
		StaffID:          &staffID,
		ObservedAmountBs: &observedBs,
		DifferenceBs:     &diff,
		CreditedBs:       &pmCreditedBs,
		ReferenceLast4:   &last4,
	}
	if err := a.resolve(ctx, review, update); err != nil {
		return models.StaffReviewResult{}, err
	}

	txID, createdAt, err := a.finance.CreditPaymentPartial(ctx, customerID, review.ReferenceID, review.ReferenceType, pmCreditedBs, review.BcvRate)
	if err != nil {
		slog.Error("RejectUnderpaid credit failed", "referenceID", review.ReferenceID, "referenceType", review.ReferenceType, "error", err)
		return models.StaffReviewResult{}, err
	}
	credit := ports.BalanceCredit{TxID: txID, RefundablePMBs: pmCreditedBs, CreatedAt: createdAt}
	if err := subject.OnRejected(ctx, review.ReferenceID, review.ID, "underpaid", credit); err != nil {
		return models.StaffReviewResult{}, err
	}
	a.broadcastResolved(ctx, review, subject)
	return resultOf(review, "reject_underpaid"), nil
}

func (a *paymentReviewsApp) RejectInvalid(ctx context.Context, reviewID, staffID uuid.UUID, reason string) (models.StaffReviewResult, error) {
	if strings.TrimSpace(reason) == "" {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}
	review, subject, err := a.loadPendingSubject(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}

	update := &models.PaymentReview{
		Status:          models.ReviewRejectedInvalid,
		StaffID:         &staffID,
		RejectionReason: &reason,
	}
	if err := a.resolve(ctx, review, update); err != nil {
		return models.StaffReviewResult{}, err
	}
	if err := subject.OnRejected(ctx, review.ReferenceID, review.ID, reason, ports.BalanceCredit{}); err != nil {
		return models.StaffReviewResult{}, err
	}
	a.broadcastResolved(ctx, review, subject)
	return resultOf(review, "reject_invalid"), nil
}

func (a *paymentReviewsApp) ListActiveForStaff(ctx context.Context) ([]models.PaymentReviewBroadcast, error) {
	if a.snapshot != nil {
		return a.snapshot(ctx)
	}
	reviews, err := a.repo.ListActiveForStaff(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.PaymentReviewBroadcast, 0, len(reviews))
	for i := range reviews {
		out = append(out, genericBroadcast(&reviews[i]))
	}
	return out, nil
}

func (a *paymentReviewsApp) loadPendingSubject(ctx context.Context, reviewID uuid.UUID) (*models.PaymentReview, ports.PaymentSubject, error) {
	review, err := a.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, nil, err
	}
	if review.Status != models.ReviewPendingReview {
		return nil, nil, errcustom.ErrInvalidStatus
	}
	subject, err := a.registry.Get(review.ReferenceType)
	if err != nil {
		return nil, nil, err
	}
	return review, subject, nil
}

func (a *paymentReviewsApp) resolve(ctx context.Context, review *models.PaymentReview, update *models.PaymentReview) error {
	ok, err := a.repo.ResolvePending(ctx, review.ID, update)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidStatus
	}
	applyResolution(review, update)
	return nil
}

func applyResolution(review *models.PaymentReview, update *models.PaymentReview) {
	review.Status = update.Status
	review.StaffID = update.StaffID
	if update.ObservedAmountBs != nil {
		review.ObservedAmountBs = update.ObservedAmountBs
	}
	if update.DifferenceBs != nil {
		review.DifferenceBs = update.DifferenceBs
	}
	if update.RejectionReason != nil {
		review.RejectionReason = update.RejectionReason
	}
	if update.CreditedBs != nil {
		review.CreditedBs = update.CreditedBs
	}
	if update.ReferenceLast4 != nil {
		review.ReferenceLast4 = update.ReferenceLast4
	}
}

func (a *paymentReviewsApp) broadcastResolved(ctx context.Context, review *models.PaymentReview, subject ports.PaymentSubject) {
	if a.hub == nil {
		return
	}
	if fresh, err := a.repo.GetByID(ctx, review.ID); err == nil {
		review = fresh
	}
	b := genericBroadcast(review)
	if enricher, ok := subject.(ports.StaffReviewEnricher); ok {
		enriched, err := enricher.EnrichStaffReview(ctx, *review)
		if err != nil {
			slog.Error("staff review enrich failed", "reviewID", review.ID, "error", err)
		} else {
			b = enriched
		}
	}
	if err := a.hub.BroadcastReview(b); err != nil {
		slog.Error("staff review broadcast failed", "reviewID", review.ID, "error", err)
	}
}

func resultOf(review *models.PaymentReview, action string) models.StaffReviewResult {
	return models.StaffReviewResult{
		ReviewID:      review.ID,
		ReferenceID:   review.ReferenceID,
		ReferenceType: review.ReferenceType,
		Action:        action,
	}
}

// prepareReferenceLast4 validates the format and, when last4 was already
// consumed, auto-rejects the review as rejected_invalid without persisting last4.
func (a *paymentReviewsApp) prepareReferenceLast4(
	ctx context.Context,
	reviewID, staffID uuid.UUID,
	referenceLast4 string,
) (last4 string, duplicate *models.StaffReviewResult, err error) {
	last4 = strings.TrimSpace(referenceLast4)
	if !referenceLast4Pattern.MatchString(last4) {
		return "", nil, errcustom.ErrInvalidStatus
	}

	exists, err := a.repo.ExistsResolvedReferenceLast4(ctx, last4)
	if err != nil {
		return "", nil, err
	}
	if !exists {
		return last4, nil, nil
	}

	reason := fmt.Sprintf("comprobante duplicado (referencia %s)", last4)
	rejected, err := a.RejectInvalid(ctx, reviewID, staffID, reason)
	if err != nil {
		return "", nil, err
	}
	rejected.DuplicateReject = true
	rejected.RejectionReason = reason
	return last4, &rejected, nil
}

var _ ports.PaymentReviewsApp = (*paymentReviewsApp)(nil)
