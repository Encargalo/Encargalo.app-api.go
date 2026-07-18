package ports

import (
	"context"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type CustomerFinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (float64, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.Transaction) (float64, error)
	ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error)
}

type CustomerFinanceApp interface {
	GetBalance(ctx context.Context, customerID uuid.UUID) (dto.CustomerBalanceResponse, error)
	GetBalanceUSD(ctx context.Context, customerID uuid.UUID) (float64, error)
	CreditPaymentOverage(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error
	CreditPaymentPartial(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error
	// CreditOrderRejected acredita el total de la orden al balance cuando el local la rechaza.
	CreditOrderRejected(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error
	DebitForOrder(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error
	RefundBalancePayment(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error
}
