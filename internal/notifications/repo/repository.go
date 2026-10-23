package repo

import (
	"context"
	"database/sql"

	"Encargalo.app-api.go/internal/notifications/domain/models"
	"github.com/uptrace/bun"
)

type Repository interface {
	SearchOrderByID(ctx context.Context, orderID string) (models.Order, error)
	SearchBalanceRefundRequestByID(ctx context.Context, requestID string) (models.BalanceRefundRequest, string, error)
	SearchPaymentReviewByID(ctx context.Context, reviewID string) (models.PaymentReview, error)
	SearchTripByID(ctx context.Context, tripID string) (models.Trip, error)
	SearchWithdrawalRequestByID(ctx context.Context, requestID string) (models.WithdrawalRequest, error)
}

type repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) Repository {
	return &repository{db}
}

func (r *repository) SearchOrderByID(ctx context.Context, orderID string) (models.Order, error) {

	var order models.Order

	if err := r.db.NewSelect().
		Model(&order).
		Relation("Shop").
		Relation("Shop.Owner").
		Relation("Customer").
		Where("?TableAlias.id = ?", orderID).
		Scan(ctx); err != nil {

		if err == sql.ErrNoRows {
			return models.Order{}, nil
		}
		return models.Order{}, err
	}

	return order, nil
}

func (r *repository) SearchBalanceRefundRequestByID(ctx context.Context, requestID string) (models.BalanceRefundRequest, string, error) {
	var req models.BalanceRefundRequest
	if err := r.db.NewSelect().
		Model(&req).
		Relation("Customer").
		Where("?TableAlias.id = ?", requestID).
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return models.BalanceRefundRequest{}, "", nil
		}
		return models.BalanceRefundRequest{}, "", err
	}

	orderStatus := ""
	if req.OrderID != nil {
		var row models.OrderStatusRow
		err := r.db.NewSelect().
			Model(&row).
			Column("status").
			Where("id = ?", *req.OrderID).
			Scan(ctx)
		if err != nil && err != sql.ErrNoRows {
			return models.BalanceRefundRequest{}, "", err
		}
		orderStatus = row.Status
	}

	return req, orderStatus, nil
}

func (r *repository) SearchPaymentReviewByID(ctx context.Context, reviewID string) (models.PaymentReview, error) {
	var review models.PaymentReview
	if err := r.db.NewSelect().
		Model(&review).
		Where("?TableAlias.id = ?", reviewID).
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return models.PaymentReview{}, nil
		}
		return models.PaymentReview{}, err
	}
	return review, nil
}

func (r *repository) SearchTripByID(ctx context.Context, tripID string) (models.Trip, error) {
	var trip models.Trip
	if err := r.db.NewSelect().
		Model(&trip).
		Relation("Customer").
		Where("?TableAlias.id = ?", tripID).
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return models.Trip{}, nil
		}
		return models.Trip{}, err
	}
	return trip, nil
}

func (r *repository) SearchWithdrawalRequestByID(ctx context.Context, requestID string) (models.WithdrawalRequest, error) {
	var req models.WithdrawalRequest
	if err := r.db.NewSelect().
		Model(&req).
		Relation("Rider").
		Where("?TableAlias.id = ?", requestID).
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return models.WithdrawalRequest{}, nil
		}
		return models.WithdrawalRequest{}, err
	}

	return req, nil
}
