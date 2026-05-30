package ports

import (
	"context"

	"Encargalo.app-api.go/internal/payment_methods/domain/models"
)

type PaymentMethodsApp interface {
	GetActivePaymentMethods(ctx context.Context) ([]models.PaymentMethod, error)
}

type PaymentMethodsRepo interface {
	GetActivePaymentMethods(ctx context.Context) ([]models.PaymentMethod, error)
}
