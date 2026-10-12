package app

import (
	"Encargalo.app-api.go/internal/pkg/id"
	"context"
	"strings"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/finance/domain/dtos"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	"Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

type staffFinanceApp struct {
	repo     ports.StaffFinanceRepo
	bcv      bcvPorts.BcvApp
	uploader uploadimage.UploadImage
}

func NewStaffFinanceApp(repo ports.StaffFinanceRepo, bcv bcvPorts.BcvApp, uploader uploadimage.UploadImage) ports.StaffFinanceApp {
	return &staffFinanceApp{repo: repo, bcv: bcv, uploader: uploader}
}

func (a *staffFinanceApp) ListPayables(ctx context.Context, entityType string) (dtos.PayablesResponse, error) {
	rows, err := a.repo.ListPayables(ctx, entityType)
	if err != nil {
		return dtos.PayablesResponse{}, err
	}
	spotRate := a.spotBcvRate(ctx)
	items := make([]dtos.PayableItem, 0, len(rows))
	for _, row := range rows {
		currency := money.NormalizeCurrency(row.Currency, financeModels.NativeCurrency(entityType))
		balanceBs, balanceUSD := money.BalanceDual(row.Amount, currency, spotRate)
		item := dtos.PayableItem{
			EntityID:   row.EntityID.String(),
			Name:       row.Name,
			BalanceBs:  balanceBs,
			BalanceUSD: balanceUSD,
			BcvRate:    spotRate,
			UpdatedAt:  row.UpdatedAt.Format(time.RFC3339),
		}
		if row.SurName != nil {
			item.SurName = *row.SurName
		}
		if row.Phone != nil {
			item.Phone = *row.Phone
		}
		items = append(items, item)
	}
	return dtos.PayablesResponse{Items: items, Total: len(items)}, nil
}

func (a *staffFinanceApp) RecordPayout(ctx context.Context, entityID, staffID uuid.UUID, entityType string, amountBs float64, notes, proofImage string) (dtos.ActionResponse, error) {
	if amountBs <= 0 {
		return dtos.ActionResponse{}, errcustom.ErrInvalidBalanceAmount
	}
	if strings.TrimSpace(proofImage) == "" {
		return dtos.ActionResponse{}, errcustom.ErrRefundProofRequired
	}
	if a.uploader == nil {
		return dtos.ActionResponse{}, errcustom.ErrUploadingImage
	}

	bcvRate, err := a.requireBcvRate(ctx)
	if err != nil {
		return dtos.ActionResponse{}, err
	}

	nativeCurrency := financeModels.NativeCurrency(entityType)
	balanceRec, err := a.repo.GetBalance(ctx, entityID, entityType)
	if err != nil {
		return dtos.ActionResponse{}, err
	}

	delta, frozenRate := money.NativeDeltaFromStaffBs(amountBs, nativeCurrency, bcvRate, true)
	if err := money.EnsureDebitAllowed(balanceRec.Amount, delta, nativeCurrency); err != nil {
		return dtos.ActionResponse{}, err
	}

	upload, err := a.uploader.UploadImage(ctx, proofImage)
	if err != nil {
		return dtos.ActionResponse{}, errcustom.ErrUploadingImage
	}
	if err := uploadimage.ValidateCloudinaryURL(upload.SecureURL); err != nil {
		return dtos.ActionResponse{}, err
	}

	proofURL := upload.SecureURL
	tx := financeModels.Transaction{
		ID:           id.New(),
		MovementType: string(financeModels.MovementTypeStaffPayout),
		Currency:     nativeCurrency,
		BcvRate:      frozenRate,
		StaffID:      &staffID,
		Notes:        optionalNotes(notes),
		ProofURL:     &proofURL,
		CreatedAt:    time.Now(),
	}

	newAmount, err := a.repo.RecordMovement(ctx, entityID, entityType, delta, 0, tx)
	if err != nil {
		return dtos.ActionResponse{}, err
	}

	return a.toActionResponse(staffID, tx, delta, newAmount, nativeCurrency, bcvRate), nil
}

func (a *staffFinanceApp) RecordAdjustment(ctx context.Context, entityID, staffID uuid.UUID, entityType, direction string, amountBs float64, notes string) (dtos.ActionResponse, error) {
	if amountBs <= 0 {
		return dtos.ActionResponse{}, errcustom.ErrInvalidBalanceAmount
	}
	if strings.TrimSpace(notes) == "" {
		return dtos.ActionResponse{}, errcustom.ErrFinanceNotesRequired
	}

	bcvRate, err := a.requireBcvRate(ctx)
	if err != nil {
		return dtos.ActionResponse{}, err
	}

	nativeCurrency := financeModels.NativeCurrency(entityType)
	balanceRec, err := a.repo.GetBalance(ctx, entityID, entityType)
	if err != nil {
		return dtos.ActionResponse{}, err
	}

	isDebit := direction == "debit"
	delta, frozenRate := money.NativeDeltaFromStaffBs(amountBs, nativeCurrency, bcvRate, isDebit)
	if err := money.EnsureDebitAllowed(balanceRec.Amount, delta, nativeCurrency); err != nil {
		return dtos.ActionResponse{}, err
	}

	movementType := financeModels.MovementTypeAdjustmentCredit
	if isDebit {
		movementType = financeModels.MovementTypeAdjustmentDebit
	}
	notesCopy := strings.TrimSpace(notes)
	tx := financeModels.Transaction{
		ID:           id.New(),
		MovementType: string(movementType),
		Currency:     nativeCurrency,
		BcvRate:      frozenRate,
		StaffID:      &staffID,
		Notes:        &notesCopy,
		CreatedAt:    time.Now(),
	}

	newAmount, err := a.repo.RecordMovement(ctx, entityID, entityType, delta, 0, tx)
	if err != nil {
		return dtos.ActionResponse{}, err
	}

	return a.toActionResponse(staffID, tx, delta, newAmount, nativeCurrency, bcvRate), nil
}

func (a *staffFinanceApp) requireBcvRate(ctx context.Context) (float64, error) {
	if a.bcv == nil {
		return 0, errcustom.ErrBcvRateUnavailable
	}
	rate, err := a.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return 0, errcustom.ErrBcvRateUnavailable
	}
	return rate.Rate, nil
}

func (a *staffFinanceApp) spotBcvRate(ctx context.Context) float64 {
	rate, err := a.requireBcvRate(ctx)
	if err != nil {
		return 0
	}
	return rate
}

func optionalNotes(notes string) *string {
	trimmed := strings.TrimSpace(notes)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (a *staffFinanceApp) toActionResponse(staffID uuid.UUID, tx financeModels.Transaction, delta, newBalance float64, currency string, spotRate float64) dtos.ActionResponse {
	amountBs, amountUSD := money.TransactionDual(delta, currency, tx.BcvRate, spotRate)
	balanceBs, balanceUSD := money.BalanceDual(newBalance, currency, spotRate)
	notes := ""
	if tx.Notes != nil {
		notes = *tx.Notes
	}
	return dtos.ActionResponse{
		TransactionID: tx.ID.String(),
		AmountBs:      amountBs,
		AmountUSD:     amountUSD,
		BcvRate:       spotRate,
		BalanceBs:     balanceBs,
		BalanceUSD:    balanceUSD,
		ProofURL:      tx.ProofURL,
		Notes:         notes,
		StaffID:       staffID.String(),
		CreatedAt:     tx.CreatedAt.Format(time.RFC3339),
	}
}
