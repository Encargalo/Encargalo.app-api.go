package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/payments/domain/models"
	"github.com/google/uuid"
)

// BalanceCredit is the ledger movement payments/app already recorded before
// calling the subject. TxID is uuid.Nil when no cash credit was written
// (reject_invalid, exact approve). RefundablePMBs is the cash-refundable
// portion, which can be smaller than the review's CreditedBs when a balance
// refund is included in the total credit.
type BalanceCredit struct {
	TxID           uuid.UUID
	RefundablePMBs float64
	CreatedAt      time.Time
}

// PaymentSubject is implemented by each domain that can own a payment review.
// payments/app resolves the CAS, detects duplicate proofs and credits balance.
// The subject mutates its own entity. reviewID is the payments.reviews primary
// key (distinct from referenceID) so the subject can re-read the resolved row.
type PaymentSubject interface {
	ExpectedAmountBs(ctx context.Context, referenceID uuid.UUID) (float64, error)
	CustomerID(ctx context.Context, referenceID uuid.UUID) (uuid.UUID, error)
	OnApproved(ctx context.Context, referenceID, reviewID uuid.UUID) error
	OnOverpaid(ctx context.Context, referenceID, reviewID uuid.UUID, diffBs float64, credit BalanceCredit) error
	OnRejected(ctx context.Context, referenceID, reviewID uuid.UUID, reason string, credit BalanceCredit) error
}

// StaffReviewEnricher is optional. A subject implements it so the resolution
// broadcast keeps domain context (shop, customer, totals) that payments/app
// cannot know.
type StaffReviewEnricher interface {
	EnrichStaffReview(ctx context.Context, review models.PaymentReview) (models.PaymentReviewBroadcast, error)
}

// PaymentSubjectRegistry resolves the PaymentSubject for a reference type.
// Callers register subjects after construction. Trips adds one Register call
// and does not edit payments/app.
type PaymentSubjectRegistry interface {
	Register(referenceType string, subject PaymentSubject)
	Get(referenceType string) (PaymentSubject, error)
}
