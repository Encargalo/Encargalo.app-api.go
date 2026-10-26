package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/finance/domain/models"
	"Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
)

const pgUniqueViolation = "23505"

// activeWithdrawalConstraint es el índice que impide más de una solicitud
// pending o processing por entidad.
const activeWithdrawalConstraint = "uq_withdrawal_requests_entity_active"

// withdrawalListMaxLimit es el techo de página del listado del CRM.
const withdrawalListMaxLimit = 50

type withdrawalRepo struct {
	db *bun.DB
}

func NewWithdrawalRepo(db *bun.DB) ports.WithdrawalRepo {
	return &withdrawalRepo{db: db}
}

// El ciclo usa una sola fila de reserva en finance.transactions por solicitud,
// cuyo movement_type se muta en cada transición — igual que los reembolsos
// (ver migración 00050). El débito ocurre una vez, al retener; completar ajusta
// el monto de esa fila a lo que realmente se transfirió y devuelve al balance
// la parte reservada que no se usó.
func (r *withdrawalRepo) Create(ctx context.Context, params ports.CreateWithdrawalParams) (models.WithdrawalRequest, error) {
	var req models.WithdrawalRequest

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.Balance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", params.EntityID, params.EntityType).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("withdrawal Create: select balance for update failed", "entityID", params.EntityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if b.Amount < params.MinAmount {
			return errcustom.ErrInsufficientBalance
		}

		held := b.Amount
		now := time.Now()
		rate := params.BcvRate
		req = models.WithdrawalRequest{
			ID:           id.New(),
			EntityID:     params.EntityID,
			EntityType:   params.EntityType,
			Amount:       held,
			Currency:     params.Currency,
			BcvRate:      &rate,
			Status:       models.WithdrawalPending,
			PMPhone:      &params.PMPhone,
			PMBankCode:   &params.PMBankCode,
			PMNationalID: &params.PMNationalID,
			CreatedAt:    now,
		}

		if _, err = tx.NewInsert().Model(&req).Exec(ctx); err != nil {
			var pgErr pgdriver.Error
			if errors.As(err, &pgErr) && pgErr.Field('C') == pgUniqueViolation && pgErr.Field('n') == activeWithdrawalConstraint {
				return errcustom.ErrWithdrawalAlreadyRequested
			}
			slog.Error("withdrawal Create: insert request failed", "entityID", params.EntityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		refType := models.ReferenceTypeWithdrawalRequest
		movement := models.Transaction{
			ID:            id.New(),
			EntityID:      params.EntityID,
			EntityType:    params.EntityType,
			ReferenceID:   &req.ID,
			ReferenceType: &refType,
			Amount:        -held,
			Currency:      params.Currency,
			BcvRate:       &rate,
			MovementType:  string(models.MovementTypeWithdrawalHold),
			CreatedAt:     now,
		}
		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("withdrawal Create: insert hold failed", "entityID", params.EntityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		return upsertBalance(ctx, tx, params.EntityID, params.EntityType, b.Amount-held, params.Currency)
	})
	if err != nil {
		return models.WithdrawalRequest{}, err
	}

	return req, nil
}

func (r *withdrawalRepo) GetByID(ctx context.Context, requestID uuid.UUID) (models.WithdrawalRequest, error) {
	var req models.WithdrawalRequest
	err := r.db.NewSelect().Model(&req).Where("id = ?", requestID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.WithdrawalRequest{}, errcustom.ErrWithdrawalNotFound
		}
		slog.Error("withdrawal GetByID failed", "requestID", requestID, "error", err)
		return models.WithdrawalRequest{}, errcustom.ErrUnexpectedError
	}
	return req, nil
}

func (r *withdrawalRepo) GetActive(ctx context.Context, entityID uuid.UUID, entityType string) (*models.WithdrawalRequest, error) {
	var req models.WithdrawalRequest
	err := r.db.NewSelect().
		Model(&req).
		Where("entity_id = ? AND entity_type = ? AND status IN (?)", entityID, entityType,
			bun.In([]models.WithdrawalStatus{models.WithdrawalPending, models.WithdrawalProcessing})).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("withdrawal GetActive failed", "entityID", entityID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &req, nil
}

func (r *withdrawalRepo) LastCompletedAt(ctx context.Context, entityID uuid.UUID, entityType string) (*time.Time, error) {
	var req models.WithdrawalRequest
	err := r.db.NewSelect().
		Model(&req).
		Column("completed_at").
		Where("entity_id = ? AND entity_type = ? AND status = ?", entityID, entityType, models.WithdrawalCompleted).
		OrderExpr("completed_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("withdrawal LastCompletedAt failed", "entityID", entityID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return req.CompletedAt, nil
}

// List pagina porque los estados cerrados (completed, rejected) crecen una fila
// por retiro y sin techo: el listado activo es corto, el histórico no.
func (r *withdrawalRepo) List(ctx context.Context, entityType string, statuses []models.WithdrawalStatus, page, limit int) ([]models.WithdrawalListRow, int, error) {
	if limit > withdrawalListMaxLimit {
		limit = withdrawalListMaxLimit
	}
	offset := (page - 1) * limit

	filter := func(q *bun.SelectQuery) *bun.SelectQuery {
		q = q.Where("w.entity_type = ?", entityType)
		if len(statuses) > 0 {
			q = q.Where("w.status IN (?)", bun.In(statuses))
		}
		return q
	}

	total, err := filter(r.db.NewSelect().TableExpr("finance.withdrawal_requests AS w")).Count(ctx)
	if err != nil {
		slog.Error("withdrawal List count failed", "entityType", entityType, "error", err)
		return nil, 0, errcustom.ErrUnexpectedError
	}

	var rows []models.WithdrawalListRow
	q := filter(r.db.NewSelect().
		TableExpr("finance.withdrawal_requests AS w").
		ColumnExpr("w.*").
		ColumnExpr("a.name").
		ColumnExpr("a.sur_name").
		ColumnExpr("a.phone").
		ColumnExpr("COALESCE(b.amount, 0) AS available_balance").
		Join("JOIN users.accounts AS a ON a.id = w.entity_id").
		Join("LEFT JOIN finance.balances AS b ON b.entity_id = w.entity_id AND b.entity_type = w.entity_type")).
		OrderExpr("w.created_at DESC").
		Limit(limit).
		Offset(offset)

	if err := q.Scan(ctx, &rows); err != nil {
		slog.Error("withdrawal List failed", "entityType", entityType, "error", err)
		return nil, 0, errcustom.ErrUnexpectedError
	}
	return rows, total, nil
}

// Claim congela el neto pagable. Si ese neto no es positivo, la solicitud se
// anula acá mismo devolviendo la reserva: el rider se queda con su deuda real
// en el balance y el cooldown no se activa, porque no se le pagó nada.
func (r *withdrawalRepo) Claim(ctx context.Context, requestID, staffID uuid.UUID) (models.WithdrawalRequest, error) {
	var out models.WithdrawalRequest
	var voided bool

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		req, err := lockRequest(ctx, tx, requestID)
		if err != nil {
			return err
		}
		if !req.CanTransitionTo(models.WithdrawalProcessing) {
			return errcustom.ErrWithdrawalInvalidTransition
		}

		available, err := lockBalanceAmount(ctx, tx, req.EntityID, req.EntityType)
		if err != nil {
			return err
		}

		payable := money.RoundBs(models.PayableFrom(req.Amount, available))
		now := time.Now()

		if payable <= 0 {
			reason := "anulada automáticamente: el efectivo cobrado por el rider consumió el monto del retiro"
			if err = releaseReservation(ctx, tx, &req, available, &staffID, &reason, now); err != nil {
				return err
			}
			// Devolver el error acá haría rollback del anulado: bun revierte la tx
			// con cualquier error del callback. El sentinel viaja después del commit.
			voided = true
			return nil
		}

		req.Status = models.WithdrawalProcessing
		req.PayableAmount = &payable
		req.StaffID = &staffID
		req.ProcessingAt = &now

		if _, err = tx.NewUpdate().
			Model(&req).
			Column("status", "payable_amount", "staff_id", "processing_at").
			WherePK().
			Exec(ctx); err != nil {
			slog.Error("withdrawal Claim: update request failed", "requestID", requestID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if err = relabelReservation(ctx, tx, req.ID, models.MovementTypeWithdrawalProcessing); err != nil {
			return err
		}

		out = req
		return nil
	})
	if err != nil {
		return models.WithdrawalRequest{}, err
	}
	if voided {
		return models.WithdrawalRequest{}, errcustom.ErrWithdrawalNothingPayable
	}

	return out, nil
}

// Complete registra el pago por el monto congelado al reclamar, sin
// recalcularlo: la transferencia bancaria ya ocurrió por ese valor y
// recalcular acá haría que los libros difieran del banco. La parte de la
// reserva que no se transfirió vuelve al balance.
func (r *withdrawalRepo) Complete(ctx context.Context, requestID, staffID uuid.UUID, proofURL string, notes *string) (models.WithdrawalRequest, error) {
	var out models.WithdrawalRequest

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		req, err := lockRequest(ctx, tx, requestID)
		if err != nil {
			return err
		}
		if !req.CanTransitionTo(models.WithdrawalCompleted) {
			return errcustom.ErrWithdrawalInvalidTransition
		}
		if req.PayableAmount == nil {
			return errcustom.ErrWithdrawalInvalidTransition
		}

		available, err := lockBalanceAmount(ctx, tx, req.EntityID, req.EntityType)
		if err != nil {
			return err
		}

		paid := *req.PayableAmount
		now := time.Now()

		req.Status = models.WithdrawalCompleted
		req.StaffID = &staffID
		req.ProofURL = &proofURL
		req.CompletedAt = &now
		req.ResolvedAt = &now

		if _, err = tx.NewUpdate().
			Model(&req).
			Column("status", "staff_id", "proof_url", "completed_at", "resolved_at").
			WherePK().
			Exec(ctx); err != nil {
			slog.Error("withdrawal Complete: update request failed", "requestID", requestID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		// La fila de reserva pasa a ser el pago y su monto queda en lo que
		// efectivamente se transfirió, con el comprobante adjunto.
		if _, err = tx.NewUpdate().
			Table("finance.transactions").
			Set("movement_type = ?", string(models.MovementTypeWithdrawalPayout)).
			Set("amount = ?", -paid).
			Set("staff_id = ?", staffID).
			Set("proof_url = ?", proofURL).
			Set("notes = ?", notes).
			Where("reference_id = ? AND reference_type = ?", req.ID, models.ReferenceTypeWithdrawalRequest).
			Exec(ctx); err != nil {
			slog.Error("withdrawal Complete: update reservation failed", "requestID", requestID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		// Devuelve lo reservado y no transferido: el saldo queda reflejando
		// solo el pago real.
		released := req.Amount - paid
		if err = upsertBalance(ctx, tx, req.EntityID, req.EntityType, available+released, req.Currency); err != nil {
			return err
		}

		out = req
		return nil
	})
	if err != nil {
		return models.WithdrawalRequest{}, err
	}

	return out, nil
}

func (r *withdrawalRepo) Reject(ctx context.Context, requestID, staffID uuid.UUID, reason string) (models.WithdrawalRequest, error) {
	var out models.WithdrawalRequest

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		req, err := lockRequest(ctx, tx, requestID)
		if err != nil {
			return err
		}
		if !req.CanTransitionTo(models.WithdrawalRejected) {
			return errcustom.ErrWithdrawalInvalidTransition
		}

		available, err := lockBalanceAmount(ctx, tx, req.EntityID, req.EntityType)
		if err != nil {
			return err
		}

		if err = releaseReservation(ctx, tx, &req, available, &staffID, &reason, time.Now()); err != nil {
			return err
		}

		out = req
		return nil
	})
	if err != nil {
		return models.WithdrawalRequest{}, err
	}

	return out, nil
}

// releaseReservation cierra la solicitud como rechazada y devuelve al balance
// todo lo retenido. La fila de reserva queda en withdrawal_restore con monto 0:
// el retiro no movió plata.
func releaseReservation(ctx context.Context, tx bun.Tx, req *models.WithdrawalRequest, available float64, staffID *uuid.UUID, reason *string, now time.Time) error {
	req.Status = models.WithdrawalRejected
	req.StaffID = staffID
	req.RejectionReason = reason
	req.ResolvedAt = &now

	if _, err := tx.NewUpdate().
		Model(req).
		Column("status", "staff_id", "rejection_reason", "resolved_at").
		WherePK().
		Exec(ctx); err != nil {
		slog.Error("withdrawal release: update request failed", "requestID", req.ID, "error", err)
		return errcustom.ErrUnexpectedError
	}

	if _, err := tx.NewUpdate().
		Table("finance.transactions").
		Set("movement_type = ?", string(models.MovementTypeWithdrawalRestore)).
		Set("amount = ?", 0).
		Where("reference_id = ? AND reference_type = ?", req.ID, models.ReferenceTypeWithdrawalRequest).
		Exec(ctx); err != nil {
		slog.Error("withdrawal release: update reservation failed", "requestID", req.ID, "error", err)
		return errcustom.ErrUnexpectedError
	}

	return upsertBalance(ctx, tx, req.EntityID, req.EntityType, available+req.Amount, req.Currency)
}

func lockRequest(ctx context.Context, tx bun.Tx, requestID uuid.UUID) (models.WithdrawalRequest, error) {
	var req models.WithdrawalRequest
	err := tx.NewSelect().
		Model(&req).
		Where("id = ?", requestID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.WithdrawalRequest{}, errcustom.ErrWithdrawalNotFound
		}
		slog.Error("withdrawal lockRequest failed", "requestID", requestID, "error", err)
		return models.WithdrawalRequest{}, errcustom.ErrUnexpectedError
	}
	return req, nil
}

func lockBalanceAmount(ctx context.Context, tx bun.Tx, entityID uuid.UUID, entityType string) (float64, error) {
	var b models.Balance
	err := tx.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		For("UPDATE").
		Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.Error("withdrawal lockBalanceAmount failed", "entityID", entityID, "error", err)
		return 0, errcustom.ErrUnexpectedError
	}
	return b.Amount, nil
}

func relabelReservation(ctx context.Context, tx bun.Tx, requestID uuid.UUID, movement models.MovementType) error {
	if _, err := tx.NewUpdate().
		Table("finance.transactions").
		Set("movement_type = ?", string(movement)).
		Where("reference_id = ? AND reference_type = ?", requestID, models.ReferenceTypeWithdrawalRequest).
		Exec(ctx); err != nil {
		slog.Error("withdrawal relabelReservation failed", "requestID", requestID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func upsertBalance(ctx context.Context, tx bun.Tx, entityID uuid.UUID, entityType string, amount float64, currency string) error {
	if currency == "" {
		currency = models.NativeCurrency(entityType)
	}
	if _, err := tx.NewInsert().
		Model(&models.Balance{
			EntityID:   entityID,
			EntityType: entityType,
			Amount:     money.RoundBs(amount),
			Currency:   currency,
		}).
		On("CONFLICT (entity_id, entity_type) DO UPDATE").
		Set("amount = EXCLUDED.amount, updated_at = now()").
		Exec(ctx); err != nil {
		slog.Error("withdrawal upsertBalance failed", "entityID", entityID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}
