package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type paymentReviewsRepo struct {
	db *bun.DB
}

func NewPaymentReviewsRepo(db *bun.DB) ports.PaymentReviewsRepo {
	return &paymentReviewsRepo{db}
}

func (r *paymentReviewsRepo) Create(ctx context.Context, review *models.PaymentReview) error {
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now()
	}
	if _, err := r.db.NewInsert().Model(review).Exec(ctx); err != nil {
		slog.Error("paymentReviews Create failed", "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *paymentReviewsRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PaymentReview, error) {
	var review models.PaymentReview
	err := r.db.NewSelect().Model(&review).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("paymentReviews GetByID failed", "id", id, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &review, nil
}

func (r *paymentReviewsRepo) ListActiveForStaff(ctx context.Context) ([]models.PaymentReview, error) {
	var reviews []models.PaymentReview
	err := r.db.NewSelect().
		Model(&reviews).
		Where("status = ?", models.ReviewPendingReview).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		slog.Error("paymentReviews ListActiveForStaff failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return reviews, nil
}

func (r *paymentReviewsRepo) ResolvePending(ctx context.Context, id uuid.UUID, update *models.PaymentReview) (bool, error) {
	now := time.Now()
	q := r.db.NewUpdate().
		Model((*models.PaymentReview)(nil)).
		Set("status = ?", update.Status).
		Set("staff_id = ?", update.StaffID).
		Set("resolved_at = ?", now).
		Where("id = ?", id).
		Where("status = ?", models.ReviewPendingReview)

	if update.ObservedAmountBs != nil {
		q = q.Set("observed_amount_bs = ?", *update.ObservedAmountBs)
	}
	if update.DifferenceBs != nil {
		q = q.Set("difference_bs = ?", *update.DifferenceBs)
	}
	if update.RejectionReason != nil {
		q = q.Set("rejection_reason = ?", *update.RejectionReason)
	}
	if update.CreditedBs != nil {
		q = q.Set("credited_bs = ?", *update.CreditedBs)
	}
	if update.ReferenceLast4 != nil {
		q = q.Set("reference_last4 = ?", *update.ReferenceLast4)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		slog.Error("paymentReviews ResolvePending failed", "id", id, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *paymentReviewsRepo) ExistsResolvedReferenceLast4(ctx context.Context, last4 string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*models.PaymentReview)(nil)).
		Where("reference_last4 = ?", last4).
		Where("status IN (?)", bun.In([]models.PaymentReviewStatus{
			models.ReviewApproved,
			models.ReviewOverpaid,
			models.ReviewRejectedUnderpaid,
		})).
		Exists(ctx)
	if err != nil {
		slog.Error("paymentReviews ExistsResolvedReferenceLast4 failed", "last4", last4, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	return exists, nil
}
