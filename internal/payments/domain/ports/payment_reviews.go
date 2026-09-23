package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/payments/domain/models"
	"github.com/google/uuid"
)

type PaymentReviewsRepo interface {
	Create(ctx context.Context, review *models.PaymentReview) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PaymentReview, error)
	ListActiveForStaff(ctx context.Context) ([]models.PaymentReview, error)
	// ResolvePending atomically updates a pending_review row. Returns false if already resolved.
	ResolvePending(ctx context.Context, id uuid.UUID, update *models.PaymentReview) (bool, error)
	// ExistsResolvedReferenceLast4 reports whether last4 is already used by an
	// approved / overpaid / rejected_underpaid review. The check is global
	// across reference types so a proof cannot be reused between an order and a trip.
	ExistsResolvedReferenceLast4(ctx context.Context, last4 string) (bool, error)
	// ListPendingForAlert filters by referenceType when non-empty.
	ListPendingForAlert(ctx context.Context, referenceType string, alertBefore time.Time) ([]models.PaymentReview, error)
	MarkAlerted(ctx context.Context, id uuid.UUID, alertedAt, alertBefore time.Time) (bool, error)
	// UpdateCreditedBs ajusta el crédito persistido cuando el subject suma el
	// balance reembolsado, que payments/app no conoce al resolver el CAS.
	UpdateCreditedBs(ctx context.Context, id uuid.UUID, creditedBs float64) error
}

type ManagerConn interface {
	Send(msg []byte)
	Close()
}

type StaffReviewHub interface {
	Register(ctx context.Context, staffID uuid.UUID, conn ManagerConn) error
	Unregister(staffID uuid.UUID, conn ManagerConn)
	BroadcastReview(review models.PaymentReviewBroadcast) error
	// SetSnapshotProvider appends a snapshot builder. Each domain registers its
	// own so a later registration does not replace the previous ones.
	SetSnapshotProvider(fn func(ctx context.Context) ([]models.PaymentReviewBroadcast, error))
}

type PaymentReviewsApp interface {
	CreateReview(ctx context.Context, referenceType string, referenceID uuid.UUID, proofURL string, expectedBs, bcvRate float64) (*models.PaymentReview, error)
	Approve(ctx context.Context, reviewID, staffID uuid.UUID, referenceLast4 string) (models.StaffReviewResult, error)
	ApproveOverpay(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error)
	RejectUnderpaid(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error)
	RejectInvalid(ctx context.Context, reviewID, staffID uuid.UUID, reason string) (models.StaffReviewResult, error)
	ListActiveForStaff(ctx context.Context) ([]models.PaymentReviewBroadcast, error)
	SetSnapshotProvider(fn func(ctx context.Context) ([]models.PaymentReviewBroadcast, error))
}
