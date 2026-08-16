package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
)

type FinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (float64, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, tx models.Transaction) (float64, error)
	ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error)
	GetLastWithdrawal(ctx context.Context, entityID uuid.UUID, entityType string) (*time.Time, error)
}

type FinanceApp interface {
	// RecordRideEarning recibe deliveryFee en USD y bcvRate para convertir y
	// acreditar en Bs (moneda nativa del balance de riders).
	RecordRideEarning(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, totalPrice, bcvRate float64, method string) error
	GetBalance(ctx context.Context, riderID uuid.UUID) (dtos.BalanceResponse, error)
	RequestWithdrawal(ctx context.Context, riderID uuid.UUID) (dtos.WithdrawalResponse, error)
	ListTransactions(ctx context.Context, riderID uuid.UUID, page, limit int) (dtos.TransactionsResponse, error)
}
