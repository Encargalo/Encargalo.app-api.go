package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/finance/domain/dtos"
	"Encargalo.app-api.go/internal/finance/domain/models"
	"github.com/google/uuid"
)

// CreateWithdrawalParams son los datos que el caller resolvió antes de crear
// la solicitud: el mínimo exigible, la tasa y el destino de pago congelado.
type CreateWithdrawalParams struct {
	EntityID     uuid.UUID
	EntityType   string
	MinAmount    float64
	Currency     string
	BcvRate      float64
	PMPhone      string
	PMBankCode   string
	PMNationalID string
}

type WithdrawalRepo interface {
	// Create retiene el saldo y crea la solicitud pending en una sola
	// transacción. Falla con ErrInsufficientBalance si el saldo no alcanza el
	// mínimo y con ErrWithdrawalAlreadyRequested si ya hay una activa.
	Create(ctx context.Context, params CreateWithdrawalParams) (models.WithdrawalRequest, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.WithdrawalRequest, error)
	// GetActive devuelve la solicitud pending o processing de la entidad, o
	// nil si no hay ninguna.
	GetActive(ctx context.Context, entityID uuid.UUID, entityType string) (*models.WithdrawalRequest, error)
	// LastCompletedAt es el ancla del cooldown: la fecha del último retiro
	// efectivamente pagado, no de la última solicitud.
	LastCompletedAt(ctx context.Context, entityID uuid.UUID, entityType string) (*time.Time, error)
	// List pagina: los estados cerrados crecen sin techo.
	List(ctx context.Context, entityType string, statuses []models.WithdrawalStatus, page, limit int) ([]models.WithdrawalListRow, int, error)
	// Claim congela el neto pagable y pasa la solicitud a processing. Si el
	// neto es <= 0 anula la solicitud y falla con ErrWithdrawalNothingPayable.
	Claim(ctx context.Context, id, staffID uuid.UUID) (models.WithdrawalRequest, error)
	// Complete registra el pago por el monto congelado al reclamar, sin
	// recalcular: la transferencia ya ocurrió por ese valor.
	Complete(ctx context.Context, id, staffID uuid.UUID, proofURL string, notes *string) (models.WithdrawalRequest, error)
	// Reject devuelve el saldo retenido y cierra la solicitud.
	Reject(ctx context.Context, id, staffID uuid.UUID, reason string) (models.WithdrawalRequest, error)
}

type WithdrawalApp interface {
	List(ctx context.Context, entityType string, statuses []models.WithdrawalStatus, page, limit int) (dtos.WithdrawalsResponse, error)
	Claim(ctx context.Context, id, staffID uuid.UUID) (dtos.ClaimResponse, error)
	Complete(ctx context.Context, id, staffID uuid.UUID, req dtos.CompleteWithdrawalRequest) (dtos.WithdrawalItem, error)
	Reject(ctx context.Context, id, staffID uuid.UUID, req dtos.RejectWithdrawalRequest) (dtos.WithdrawalItem, error)
}
