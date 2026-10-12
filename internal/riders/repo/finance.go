package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/id"
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

func (r *financeRepo) GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.Balance, error) {
	var b models.Balance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Balance{Currency: models.CurrencyVES}, nil
		}
		slog.Error("GetBalance: failed", "entityID", entityID, "error", err)
		return models.Balance{}, errcustom.ErrUnexpectedError
	}
	if b.Currency == "" {
		b.Currency = models.CurrencyVES
	}
	return b, nil
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

		// La moneda del movimiento la decide el caller (app): este repo es
		// genérico por entity_type y no debe asumir una moneda fija.
		_, err = tx.NewInsert().
			Model(&models.Balance{
				EntityID:   entityID,
				EntityType: entityType,
				Amount:     newAmount,
				Currency:   movement.Currency,
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
		if movement.ID == uuid.Nil {
			movement.ID = id.New()
		}
		if movement.CreatedAt.IsZero() {
			movement.CreatedAt = time.Now()
		}

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

func (r *financeRepo) ListRecentTransactions(ctx context.Context, entityID uuid.UUID, entityType string, limit int) ([]models.Transaction, error) {
	var txs []models.Transaction
	err := r.db.NewSelect().
		Model(&txs).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		OrderExpr("created_at DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		slog.Error("ListRecentTransactions: failed", "entityID", entityID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return txs, nil
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
