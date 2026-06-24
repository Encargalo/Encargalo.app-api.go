package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type financeRepo struct {
	db *bun.DB
}

func NewFinanceRepo(db *bun.DB) ports.FinanceRepo {
	return &financeRepo{db}
}

func (r *financeRepo) GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (float64, error) {
	var b models.Balance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		slog.Error("GetBalance: failed", "entityID", entityID, "error", err)
		return 0, errcustom.ErrUnexpectedError
	}
	return b.Amount, nil
}

// RecordMovement actualiza el balance y registra el movimiento en una sola transacción.
// Si minBalance > 0, valida que el balance actual sea >= minBalance bajo el SELECT FOR UPDATE,
// evitando race conditions en operaciones de retiro.
func (r *financeRepo) RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.Transaction) (float64, error) {
	var newAmount float64

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.Balance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", entityID, entityType).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("RecordMovement: select for update failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if minBalance > 0 && b.Amount < minBalance {
			return errcustom.ErrInsufficientBalance
		}

		newAmount = b.Amount + delta

		_, err = tx.NewInsert().
			Model(&models.Balance{
				EntityID:   entityID,
				EntityType: entityType,
				Amount:     newAmount,
			}).
			On("CONFLICT (entity_id, entity_type) DO UPDATE").
			Set("amount = EXCLUDED.amount, updated_at = now()").
			Exec(ctx)
		if err != nil {
			slog.Error("RecordMovement: balance upsert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		movement.EntityID = entityID
		movement.EntityType = entityType
		movement.Amount = delta

		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("RecordMovement: transaction insert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return newAmount, nil
}

func (r *financeRepo) GetLastWithdrawal(ctx context.Context, entityID uuid.UUID, entityType string) (*time.Time, error) {
	var tx models.Transaction
	err := r.db.NewSelect().
		Model(&tx).
		Where("entity_id = ? AND entity_type = ? AND movement_type = ?", entityID, entityType, models.MovementTypeWithdraw).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("GetLastWithdrawal: failed", "entityID", entityID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &tx.CreatedAt, nil
}

func (r *financeRepo) ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error) {
	if limit > 50 {
		limit = 50
	}
	offset := (page - 1) * limit

	var txs []models.Transaction
	total, err := r.db.NewSelect().
		Model(&txs).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		OrderExpr("created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		slog.Error("ListTransactions: failed", "entityID", entityID, "error", err)
		return nil, 0, errcustom.ErrUnexpectedError
	}

	return txs, total, nil
}
