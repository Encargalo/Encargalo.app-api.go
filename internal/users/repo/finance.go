package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customerFinanceRepo struct {
	db *bun.DB
}

func NewCustomerFinanceRepo(db *bun.DB) ports.CustomerFinanceRepo {
	return &customerFinanceRepo{db}
}

func (r *customerFinanceRepo) GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (float64, error) {
	var b models.Balance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		slog.Error("customer GetBalance: failed", "entityID", entityID, "error", err)
		return 0, errcustom.ErrUnexpectedError
	}
	return b.Amount, nil
}

func (r *customerFinanceRepo) SumBalancePaymentDebit(ctx context.Context, customerID, orderID uuid.UUID) (float64, error) {
	var sum float64
	err := r.db.NewSelect().
		Model((*models.Transaction)(nil)).
		ColumnExpr("COALESCE(SUM(ABS(amount)), 0)").
		Where("entity_id = ? AND entity_type = ?", customerID, models.EntityTypeCustomer).
		Where("order_id = ?", orderID).
		Where("movement_type = ?", models.MovementTypeBalancePayment).
		Where("amount < 0").
		Scan(ctx, &sum)
	if err != nil {
		slog.Error("SumBalancePaymentDebit failed", "customerID", customerID, "orderID", orderID, "error", err)
		return 0, errcustom.ErrUnexpectedError
	}
	return sum, nil
}

func (r *customerFinanceRepo) RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.Transaction) (float64, error) {
	var newAmount float64

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.Balance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", entityID, entityType).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("customer RecordMovement: select failed", "entityID", entityID, "error", err)
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
			slog.Error("customer RecordMovement: balance upsert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		movement.EntityID = entityID
		movement.EntityType = entityType
		movement.Amount = delta
		if movement.ID == uuid.Nil {
			movement.ID = uuid.New()
		}
		// bun inserta CreatedAt=zero y pisa el DEFAULT now() de Postgres.
		if movement.CreatedAt.IsZero() {
			movement.CreatedAt = time.Now()
		}

		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("customer RecordMovement: tx insert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return newAmount, nil
}

func (r *customerFinanceRepo) ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error) {
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
		slog.Error("customer ListTransactions: failed", "entityID", entityID, "error", err)
		return nil, 0, errcustom.ErrUnexpectedError
	}

	return txs, total, nil
}

func (r *customerFinanceRepo) GetTransaction(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	var t models.Transaction
	err := r.db.NewSelect().Model(&t).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("customer GetTransaction: failed", "id", id, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &t, nil
}

func (r *customerFinanceRepo) SetTransactionRefundRequestID(ctx context.Context, txID, refundRequestID uuid.UUID) error {
	res, err := r.db.NewUpdate().
		Model((*models.Transaction)(nil)).
		Set("refund_request_id = ?", refundRequestID).
		Where("id = ?", txID).
		Exec(ctx)
	if err != nil {
		slog.Error("SetTransactionRefundRequestID failed", "txID", txID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrNotFound
	}
	return nil
}

func (r *customerFinanceRepo) GetPagoMovil(ctx context.Context, customerID uuid.UUID) (*models.CustomerPagoMovil, error) {
	var acc models.Account
	err := r.db.NewSelect().
		Model(&acc).
		Column("id", "pago_movil", "updated_at").
		Where("id = ?", customerID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("GetPagoMovil failed", "customerID", customerID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	if acc.PagoMovil == nil || acc.PagoMovil.Phone == "" {
		return nil, errcustom.ErrNotFound
	}
	acc.PagoMovil.CustomerID = customerID
	acc.PagoMovil.UpdatedAt = acc.UpdatedAt
	return acc.PagoMovil, nil
}

func (r *customerFinanceRepo) UpsertPagoMovil(ctx context.Context, pm *models.CustomerPagoMovil) error {
	details := &models.CustomerPagoMovil{
		Phone:      pm.Phone,
		BankCode:   pm.BankCode,
		NationalID: pm.NationalID,
	}
	res, err := r.db.NewUpdate().
		Model((*models.Account)(nil)).
		Set("pago_movil = ?", details).
		Set("updated_at = now()").
		Where("id = ?", pm.CustomerID).
		Exec(ctx)
	if err != nil {
		slog.Error("UpsertPagoMovil failed", "customerID", pm.CustomerID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrNotFound
	}
	return nil
}

func (r *customerFinanceRepo) CreateRefundRequest(ctx context.Context, req *models.BalanceRefundRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(req).Exec(ctx)
	if err != nil {
		slog.Error("CreateRefundRequest failed", "customerID", req.CustomerID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

// CreateRefundRequestWithDebit inserta la solicitud y debita el balance en una sola tx.
func (r *customerFinanceRepo) CreateRefundRequestWithDebit(ctx context.Context, req *models.BalanceRefundRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}

	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.Balance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", req.CustomerID, models.EntityTypeCustomer).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("CreateRefundRequestWithDebit balance select failed", "customerID", req.CustomerID, "error", err)
			return errcustom.ErrUnexpectedError
		}
		if b.Amount < req.AmountBs {
			return errcustom.ErrInsufficientBalance
		}

		if _, err = tx.NewInsert().Model(req).Exec(ctx); err != nil {
			slog.Error("CreateRefundRequestWithDebit insert request failed", "customerID", req.CustomerID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		newAmount := b.Amount - req.AmountBs
		_, err = tx.NewInsert().
			Model(&models.Balance{
				EntityID:   req.CustomerID,
				EntityType: models.EntityTypeCustomer,
				Amount:     newAmount,
				Currency:   models.CurrencyVES,
			}).
			On("CONFLICT (entity_id, entity_type) DO UPDATE").
			Set("amount = EXCLUDED.amount, updated_at = now()").
			Exec(ctx)
		if err != nil {
			slog.Error("CreateRefundRequestWithDebit balance upsert failed", "customerID", req.CustomerID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		rid := req.ID
		movement := models.Transaction{
			ID:              uuid.New(),
			EntityID:        req.CustomerID,
			EntityType:      models.EntityTypeCustomer,
			OrderID:         req.OrderID,
			Amount:          -req.AmountBs,
			Currency:        models.CurrencyVES,
			MovementType:    string(models.MovementTypeBalanceRefundHold),
			RefundRequestID: &rid,
			CreatedAt:       req.CreatedAt,
		}
		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("CreateRefundRequestWithDebit movement insert failed", "requestID", req.ID, "error", err)
			return errcustom.ErrUnexpectedError
		}
		return nil
	})
}

func (r *customerFinanceRepo) GetRefundRequest(ctx context.Context, id uuid.UUID) (*models.BalanceRefundRequest, error) {
	var req models.BalanceRefundRequest
	err := r.db.NewSelect().Model(&req).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("GetRefundRequest failed", "id", id, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &req, nil
}

func (r *customerFinanceRepo) ListRefundRequestsByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.BalanceRefundRequest, error) {
	var list []models.BalanceRefundRequest
	err := r.db.NewSelect().
		Model(&list).
		Where("customer_id = ?", customerID).
		OrderExpr("created_at DESC").
		Limit(50).
		Scan(ctx)
	if err != nil {
		slog.Error("ListRefundRequestsByCustomer failed", "customerID", customerID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return list, nil
}

func (r *customerFinanceRepo) ListActiveRefundRequests(ctx context.Context) ([]models.BalanceRefundRequest, error) {
	var list []models.BalanceRefundRequest
	err := r.db.NewSelect().
		Model(&list).
		Where("status IN (?)", bun.In([]models.RefundRequestStatus{models.RefundPending, models.RefundAccepted})).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		slog.Error("ListActiveRefundRequests failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return list, nil
}

func (r *customerFinanceRepo) HasActiveRefundForTransaction(ctx context.Context, transactionID uuid.UUID) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*models.BalanceRefundRequest)(nil)).
		Where("transaction_id = ?", transactionID).
		Where("status IN (?)", bun.In([]models.RefundRequestStatus{models.RefundPending, models.RefundAccepted, models.RefundCompleted})).
		Exists(ctx)
	if err != nil {
		slog.Error("HasActiveRefundForTransaction failed", "transactionID", transactionID, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	return exists, nil
}

func (r *customerFinanceRepo) AcceptPending(ctx context.Context, id, staffID uuid.UUID) (bool, error) {
	var accepted bool
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now()
		res, err := tx.NewUpdate().
			Model((*models.BalanceRefundRequest)(nil)).
			Set("status = ?", models.RefundAccepted).
			Set("staff_id = ?", staffID).
			Set("accepted_at = ?", now).
			Where("id = ? AND status = ?", id, models.RefundPending).
			Exec(ctx)
		if err != nil {
			slog.Error("AcceptPending failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			accepted = false
			return nil
		}

		mRes, err := tx.NewUpdate().
			Model((*models.Transaction)(nil)).
			Set("movement_type = ?", models.MovementTypeBalanceRefundProcessing).
			Where("refund_request_id = ?", id).
			Where("movement_type = ?", models.MovementTypeBalanceRefundHold).
			Exec(ctx)
		if err != nil {
			slog.Error("AcceptPending movement update failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}
		mn, _ := mRes.RowsAffected()
		if mn == 0 {
			slog.Error("AcceptPending hold movement missing", "id", id)
			return errcustom.ErrUnexpectedError
		}
		accepted = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return accepted, nil
}

func (r *customerFinanceRepo) RejectPendingWithRestore(ctx context.Context, id, staffID uuid.UUID, reason string) (bool, error) {
	var rejected bool
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var req models.BalanceRefundRequest
		err := tx.NewSelect().
			Model(&req).
			Where("id = ? AND status = ?", id, models.RefundPending).
			For("UPDATE").
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rejected = false
				return nil
			}
			slog.Error("RejectPendingWithRestore select failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}

		now := time.Now()
		q := tx.NewUpdate().
			Model((*models.BalanceRefundRequest)(nil)).
			Set("status = ?", models.RefundRejected).
			Set("staff_id = ?", staffID).
			Set("resolved_at = ?", now).
			Where("id = ? AND status = ?", id, models.RefundPending)
		if reason != "" {
			q = q.Set("rejection_reason = ?", reason)
		}
		res, err := q.Exec(ctx)
		if err != nil {
			slog.Error("RejectPendingWithRestore update failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			rejected = false
			return nil
		}

		var b models.Balance
		err = tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", req.CustomerID, models.EntityTypeCustomer).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("RejectPendingWithRestore balance select failed", "customerID", req.CustomerID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		newAmount := b.Amount + req.AmountBs
		_, err = tx.NewInsert().
			Model(&models.Balance{
				EntityID:   req.CustomerID,
				EntityType: models.EntityTypeCustomer,
				Amount:     newAmount,
				Currency:   models.CurrencyVES,
			}).
			On("CONFLICT (entity_id, entity_type) DO UPDATE").
			Set("amount = EXCLUDED.amount, updated_at = now()").
			Exec(ctx)
		if err != nil {
			slog.Error("RejectPendingWithRestore balance upsert failed", "customerID", req.CustomerID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		rid := req.ID
		movement := models.Transaction{
			ID:              uuid.New(),
			EntityID:        req.CustomerID,
			EntityType:      models.EntityTypeCustomer,
			OrderID:         req.OrderID,
			Amount:          req.AmountBs,
			Currency:        models.CurrencyVES,
			MovementType:    string(models.MovementTypeBalanceRefundRestore),
			RefundRequestID: &rid,
			CreatedAt:       now,
		}
		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("RejectPendingWithRestore movement insert failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}

		rejected = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return rejected, nil
}

func (r *customerFinanceRepo) CompleteAccepted(ctx context.Context, id uuid.UUID, proofURL string) (bool, error) {
	var completed bool
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now()
		res, err := tx.NewUpdate().
			Model((*models.BalanceRefundRequest)(nil)).
			Set("status = ?", models.RefundCompleted).
			Set("proof_url = ?", proofURL).
			Set("completed_at = ?", now).
			Set("resolved_at = ?", now).
			Where("id = ? AND status = ?", id, models.RefundAccepted).
			Exec(ctx)
		if err != nil {
			slog.Error("CompleteAccepted failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			completed = false
			return nil
		}

		mRes, err := tx.NewUpdate().
			Model((*models.Transaction)(nil)).
			Set("movement_type = ?", models.MovementTypeBalanceRefundPayout).
			Where("refund_request_id = ?", id).
			Where("movement_type = ?", models.MovementTypeBalanceRefundProcessing).
			Exec(ctx)
		if err != nil {
			slog.Error("CompleteAccepted movement update failed", "id", id, "error", err)
			return errcustom.ErrUnexpectedError
		}
		mn, _ := mRes.RowsAffected()
		if mn == 0 {
			slog.Error("CompleteAccepted processing movement missing", "id", id)
			return errcustom.ErrUnexpectedError
		}
		completed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return completed, nil
}

func (r *customerFinanceRepo) ListIncompleteForAlert(ctx context.Context, alertBefore time.Time) ([]models.BalanceRefundRequest, error) {
	var list []models.BalanceRefundRequest
	err := r.db.NewSelect().
		Model(&list).
		Where("status IN (?)", bun.In([]models.RefundRequestStatus{models.RefundPending, models.RefundAccepted})).
		Where("last_alerted_at IS NULL OR last_alerted_at <= ?", alertBefore).
		OrderExpr("created_at ASC").
		Limit(100).
		Scan(ctx)
	if err != nil {
		slog.Error("ListIncompleteForAlert failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return list, nil
}

func (r *customerFinanceRepo) MarkAlerted(ctx context.Context, id uuid.UUID, alertedAt, alertBefore time.Time) (bool, error) {
	res, err := r.db.NewUpdate().
		Model((*models.BalanceRefundRequest)(nil)).
		Set("last_alerted_at = ?", alertedAt).
		Where("id = ?", id).
		Where("status IN (?)", bun.In([]models.RefundRequestStatus{models.RefundPending, models.RefundAccepted})).
		Where("last_alerted_at IS NULL OR last_alerted_at <= ?", alertBefore).
		Exec(ctx)
	if err != nil {
		slog.Error("MarkAlerted failed", "id", id, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
