package ports

import (
	"context"

	"Encargalo.app-api.go/internal/finance/domain/dtos"
	"Encargalo.app-api.go/internal/finance/domain/models"
	"github.com/google/uuid"
)

type StaffFinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.Balance, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, tx models.Transaction) (float64, error)
	ListPayables(ctx context.Context, entityType string) ([]models.PayableRow, error)
}

type StaffFinanceApp interface {
	ListPayables(ctx context.Context, entityType string) (dtos.PayablesResponse, error)
	RecordPayout(ctx context.Context, entityID, staffID uuid.UUID, entityType string, amountBs float64, notes, proofImage string) (dtos.ActionResponse, error)
	RecordAdjustment(ctx context.Context, entityID, staffID uuid.UUID, entityType, direction string, amountBs float64, notes string) (dtos.ActionResponse, error)
}
