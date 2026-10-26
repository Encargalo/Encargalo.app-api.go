package app

import (
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

type withdrawalApp struct {
	repo     ports.WithdrawalRepo
	bcv      bcvPorts.BcvApp
	uploader uploadimage.UploadImage
}

func NewWithdrawalApp(repo ports.WithdrawalRepo, bcv bcvPorts.BcvApp, uploader uploadimage.UploadImage) ports.WithdrawalApp {
	return &withdrawalApp{repo: repo, bcv: bcv, uploader: uploader}
}

func (a *withdrawalApp) List(ctx context.Context, entityType string, statuses []financeModels.WithdrawalStatus, page, limit int) (dtos.WithdrawalsResponse, error) {
	rows, total, err := a.repo.List(ctx, entityType, statuses, page, limit)
	if err != nil {
		return dtos.WithdrawalsResponse{}, err
	}

	spotRate := a.spotRate(ctx)
	items := make([]dtos.WithdrawalItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toWithdrawalItem(row, entityType, spotRate))
	}

	return dtos.WithdrawalsResponse{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (a *withdrawalApp) Claim(ctx context.Context, requestID, staffID uuid.UUID) (dtos.ClaimResponse, error) {
	req, err := a.repo.Claim(ctx, requestID, staffID)
	if err != nil {
		return dtos.ClaimResponse{}, err
	}

	spotRate := a.spotRate(ctx)
	currency := money.NormalizeCurrency(req.Currency, financeModels.NativeCurrency(req.EntityType))
	payable := 0.0
	if req.PayableAmount != nil {
		payable = *req.PayableAmount
	}
	payableBs, payableUSD := money.BalanceDual(payable, currency, spotRate)

	resp := dtos.ClaimResponse{
		ID:         req.ID.String(),
		PayableBs:  payableBs,
		PayableUSD: payableUSD,
		BcvRate:    spotRate,
		Status:     string(req.Status),
	}
	if req.ProcessingAt != nil {
		resp.ClaimedAt = req.ProcessingAt.Format(time.RFC3339)
	}
	if req.PMPhone != nil {
		resp.PMPhone = *req.PMPhone
	}
	if req.PMBankCode != nil {
		resp.PMBankCode = *req.PMBankCode
	}
	if req.PMNationalID != nil {
		resp.PMNationalID = *req.PMNationalID
	}
	return resp, nil
}

func (a *withdrawalApp) Complete(ctx context.Context, requestID, staffID uuid.UUID, req dtos.CompleteWithdrawalRequest) (dtos.WithdrawalItem, error) {
	if strings.TrimSpace(req.ProofImage) == "" {
		return dtos.WithdrawalItem{}, errcustom.ErrWithdrawalProofRequired
	}
	if a.uploader == nil {
		return dtos.WithdrawalItem{}, errcustom.ErrUploadingImage
	}

	// Se valida el estado antes de subir el comprobante: no tiene sentido
	// dejar una imagen huérfana en Cloudinary por una solicitud que no se
	// puede completar.
	current, err := a.repo.GetByID(ctx, requestID)
	if err != nil {
		return dtos.WithdrawalItem{}, err
	}
	if !current.CanTransitionTo(financeModels.WithdrawalCompleted) {
		return dtos.WithdrawalItem{}, errcustom.ErrWithdrawalInvalidTransition
	}

	upload, err := a.uploader.UploadImage(ctx, req.ProofImage)
	if err != nil {
		return dtos.WithdrawalItem{}, errcustom.ErrUploadingImage
	}
	if err = uploadimage.ValidateCloudinaryURL(upload.SecureURL); err != nil {
		return dtos.WithdrawalItem{}, err
	}

	updated, err := a.repo.Complete(ctx, requestID, staffID, upload.SecureURL, optionalNotes(req.Notes))
	if err != nil {
		return dtos.WithdrawalItem{}, err
	}

	return a.toItem(ctx, updated), nil
}

func (a *withdrawalApp) Reject(ctx context.Context, requestID, staffID uuid.UUID, req dtos.RejectWithdrawalRequest) (dtos.WithdrawalItem, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return dtos.WithdrawalItem{}, errcustom.ErrWithdrawalReasonRequired
	}

	updated, err := a.repo.Reject(ctx, requestID, staffID, strings.TrimSpace(req.Reason))
	if err != nil {
		return dtos.WithdrawalItem{}, err
	}

	return a.toItem(ctx, updated), nil
}

func (a *withdrawalApp) toItem(ctx context.Context, req financeModels.WithdrawalRequest) dtos.WithdrawalItem {
	return toWithdrawalItem(financeModels.WithdrawalListRow{WithdrawalRequest: req}, req.EntityType, a.spotRate(ctx))
}

func (a *withdrawalApp) spotRate(ctx context.Context) float64 {
	if a.bcv == nil {
		return 0
	}
	rate, err := a.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return 0
	}
	return rate.Rate
}

func toWithdrawalItem(row financeModels.WithdrawalListRow, entityType string, spotRate float64) dtos.WithdrawalItem {
	currency := money.NormalizeCurrency(row.Currency, financeModels.NativeCurrency(entityType))
	amountBs, amountUSD := money.BalanceDual(row.Amount, currency, spotRate)
	payableBs, payableUSD := money.BalanceDual(row.Payable(), currency, spotRate)

	item := dtos.WithdrawalItem{
		ID:              row.ID.String(),
		EntityID:        row.EntityID.String(),
		Name:            row.Name,
		AmountBs:        amountBs,
		AmountUSD:       amountUSD,
		PayableBs:       payableBs,
		PayableUSD:      payableUSD,
		PayableFrozen:   row.PayableAmount != nil,
		BcvRate:         spotRate,
		Status:          string(row.Status),
		ProofURL:        row.ProofURL,
		RejectionReason: row.RejectionReason,
		CreatedAt:       row.CreatedAt.Format(time.RFC3339),
	}
	if row.SurName != nil {
		item.SurName = *row.SurName
	}
	if row.Phone != nil {
		item.Phone = *row.Phone
	}
	if row.PMPhone != nil {
		item.PMPhone = *row.PMPhone
	}
	if row.PMBankCode != nil {
		item.PMBankCode = *row.PMBankCode
	}
	if row.PMNationalID != nil {
		item.PMNationalID = *row.PMNationalID
	}
	if row.StaffID != nil {
		s := row.StaffID.String()
		item.StaffID = &s
	}
	if row.ProcessingAt != nil {
		s := row.ProcessingAt.Format(time.RFC3339)
		item.ProcessingAt = &s
	}
	if row.CompletedAt != nil {
		s := row.CompletedAt.Format(time.RFC3339)
		item.CompletedAt = &s
	}
	if row.ResolvedAt != nil {
		s := row.ResolvedAt.Format(time.RFC3339)
		item.ResolvedAt = &s
	}
	return item
}
