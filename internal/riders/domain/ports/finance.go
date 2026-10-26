package ports

import (
	"context"
	"time"

	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	financePorts "Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type FinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.Balance, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, tx models.Transaction) (float64, error)
	ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error)
	// ListRecentTransactions: como ListTransactions pero sin el COUNT(*) que GetBalance nunca usa.
	ListRecentTransactions(ctx context.Context, entityID uuid.UUID, entityType string, limit int) ([]models.Transaction, error)
}

// WithdrawalStore es la porción de la persistencia de retiros que riders
// necesita: crear la solicitud y responder si puede crearse. Las transiciones
// (reclamar, pagar, rechazar) son de staff y viven en finance.
type WithdrawalStore interface {
	Create(ctx context.Context, params financePorts.CreateWithdrawalParams) (financeModels.WithdrawalRequest, error)
	GetActive(ctx context.Context, entityID uuid.UUID, entityType string) (*financeModels.WithdrawalRequest, error)
	// LastCompletedAt ancla el cooldown en el último retiro pagado, no en la
	// última solicitud: lo que la regla acota es la transferencia manual.
	LastCompletedAt(ctx context.Context, entityID uuid.UUID, entityType string) (*time.Time, error)
}

// PaymentDetailsStore es la porción de users que riders necesita: el Pago
// Móvil de la cuenta, que es el destino del retiro. Devuelve ErrNotFound
// cuando el rider no lo registró.
type PaymentDetailsStore interface {
	GetPagoMovil(ctx context.Context, userID uuid.UUID) (*usersModels.CustomerPagoMovil, error)
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
	RecordRideCashCollected(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, cashUSD, bcvRate float64) error
	RecordTripCashCollected(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, cashUSD, bcvRate float64) error
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
	RecordTripCashCollected(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, cashUSD, bcvRate float64) error
}

// WithdrawalPublisher avisa al equipo de una solicitud nueva. Es conveniencia,
// no fuente de verdad: un fallo acá no debe abortar el retiro.
type WithdrawalPublisher interface {
	PublishWithdrawalRequested(ctx context.Context, req financeModels.WithdrawalRequest) error
}
