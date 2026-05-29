package app

import (
	"context"

	"Encargalo.app-api.go/internal/payment_methods/domain/models"
	"Encargalo.app-api.go/internal/payment_methods/domain/ports"
)

type paymentMethodsApp struct {
	repo ports.PaymentMethodsRepo
}

func NewPaymentMethodsApp(repo ports.PaymentMethodsRepo) ports.PaymentMethodsApp {
	return &paymentMethodsApp{repo}
}

func (a *paymentMethodsApp) GetActivePaymentMethods(ctx context.Context) ([]models.PaymentMethod, error) {
	return a.repo.GetActivePaymentMethods(ctx)
}
