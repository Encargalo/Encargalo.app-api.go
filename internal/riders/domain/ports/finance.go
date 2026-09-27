package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
)

type FinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.Balance, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, tx models.Transaction) (float64, error)
	ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error)
	// ListRecentTransactions: como ListTransactions pero sin el COUNT(*) que GetBalance nunca usa.
	ListRecentTransactions(ctx context.Context, entityID uuid.UUID, entityType string, limit int) ([]models.Transaction, error)
	GetLastWithdrawal(ctx context.Context, entityID uuid.UUID, entityType string) (*time.Time, error)
}

type FinanceApp interface {
	// RecordRideEarning recibe deliveryFee en USD y bcvRate para convertir y
	// acreditar en Bs (moneda nativa del balance de riders).
	RecordRideEarning(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, totalPrice, bcvRate float64, method string) error
	// RecordTripEarning recibe fare (tarifa del viaje) en USD y bcvRate para
	// convertir y acreditar en Bs, análogo a RecordRideEarning pero sin
	// método de pago (los viajes no tienen ese concepto todavía).
	RecordTripEarning(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, bcvRate float64) error
	// RecordTripCash debita lo que el rider cobró en efectivo y no es su pago.
	RecordTripCash(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, bcvRate float64) error
	GetBalance(ctx context.Context, riderID uuid.UUID) (dtos.BalanceResponse, error)
	RequestWithdrawal(ctx context.Context, riderID uuid.UUID) (dtos.WithdrawalResponse, error)
	ListTransactions(ctx context.Context, riderID uuid.UUID, page, limit int) (dtos.TransactionsResponse, error)
}

// TripEarningRecorder is the narrow slice of FinanceApp that the rider-trip
// matching flow needs to settle a completed trip — implemented by financeApp.
type TripEarningRecorder interface {
	RecordTripEarning(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, bcvRate float64) error
	// RecordTripCash debita lo que el rider cobró en efectivo y no es su pago.
	RecordTripCash(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, bcvRate float64) error
}
