package repo

import (
	"context"

	"Encargalo.app-api.go/internal/payment_methods/domain/models"
	"Encargalo.app-api.go/internal/payment_methods/domain/ports"
	"github.com/uptrace/bun"
)

type paymentMethodsRepo struct {
	db *bun.DB
}

func NewPaymentMethodsRepo(db *bun.DB) ports.PaymentMethodsRepo {
	return &paymentMethodsRepo{db}
}

func (r *paymentMethodsRepo) GetActivePaymentMethods(ctx context.Context) ([]models.PaymentMethod, error) {
	var methods []models.PaymentMethod
	err := r.db.NewSelect().
		Model(&methods).
		Where("is_active = TRUE").
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return methods, nil
}
