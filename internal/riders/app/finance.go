package app

import (
	"context"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

const entityTypeRider = "rider"

type financeApp struct {
	repo ports.FinanceRepo
	bcv  bcvPorts.BcvApp
}

func NewFinanceApp(repo ports.FinanceRepo, bcv bcvPorts.BcvApp) ports.FinanceApp {
	return &financeApp{repo, bcv}
}

// RecordRideEarning credits the rider's balance with 80% of the delivery fee
// (USD, calculado por orders), convertido a Bs con bcvRate — la tasa que el
// caller resolvió para este movimiento (ver orders/app.CompleteDelivery, que
// usa la tasa BCV vigente al completar la entrega, no la del quote original
// que ya expiró para entonces). Falla en vez de acreditar Bs=0 si no hay tasa.
func (f *financeApp) RecordRideEarning(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, _ float64, bcvRate float64, method string) error {
	if bcvRate <= 0 {
		return errcustom.ErrBcvRateUnavailable
	}
	deltaBs := money.RoundBs(deliveryFee * models.RiderShare * bcvRate)
	rate := bcvRate

	tx := models.Transaction{
		ID:            uuid.New(),
		OrderID:       &orderID,
		MovementType:  string(models.MovementTypeRideBank),
		PaymentMethod: &method,
		Currency:      models.CurrencyVES,
		BcvRate:       &rate,
		CreatedAt:     time.Now(),
	}

	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, deltaBs, 0, tx)
	return err
}

func (f *financeApp) GetBalance(ctx context.Context, riderID uuid.UUID) (dtos.BalanceResponse, error) {
	balance, err := f.repo.GetBalance(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.BalanceResponse{}, err
	}

	txs, _, err := f.repo.ListTransactions(ctx, riderID, entityTypeRider, 1, 10)
	if err != nil {
		return dtos.BalanceResponse{}, err
	}

	minBs := f.withdrawalMinBs(ctx)
	return dtos.BalanceResponse{
		Balance:         balance,
		Zone:            getZone(balance, minBs),
		WithdrawalMinBs: minBs,
		Transactions:    toTransactionDTOs(txs),
	}, nil
}

// withdrawalMinBs convierte el piso de negocio (WithdrawalMinUSD) a Bs con la
// tasa BCV vigente. Si la tasa no está disponible devuelve 0 (informativo;
// RequestWithdrawal sí falla explícitamente en ese caso, ver abajo).
func (f *financeApp) withdrawalMinBs(ctx context.Context) float64 {
	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return 0
	}
	return money.RoundBs(models.WithdrawalMinUSD * rate.Rate)
}

func (f *financeApp) RequestWithdrawal(ctx context.Context, riderID uuid.UUID) (dtos.WithdrawalResponse, error) {
	lastWithdrawal, err := f.repo.GetLastWithdrawal(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.WithdrawalResponse{}, err
	}
	if lastWithdrawal != nil && time.Since(*lastWithdrawal) < models.WithdrawalCooldown {
		return dtos.WithdrawalResponse{}, errcustom.ErrWithdrawalCooldown
	}

	minBs := f.withdrawalMinBs(ctx)
	if minBs <= 0 {
		return dtos.WithdrawalResponse{}, errcustom.ErrBcvRateUnavailable
	}

	balance, err := f.repo.GetBalance(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.WithdrawalResponse{}, err
	}

	tx := models.Transaction{
		ID:           uuid.New(),
		MovementType: string(models.MovementTypeWithdraw),
		Currency:     models.CurrencyVES,
		CreatedAt:    time.Now(),
	}

	if _, err = f.repo.RecordMovement(ctx, riderID, entityTypeRider, -balance, minBs, tx); err != nil {
		return dtos.WithdrawalResponse{}, err
	}

	return dtos.WithdrawalResponse{AmountWithdrawn: balance}, nil
}

func (f *financeApp) ListTransactions(ctx context.Context, riderID uuid.UUID, page, limit int) (dtos.TransactionsResponse, error) {
	if limit > 50 {
		limit = 50
	}
	txs, total, err := f.repo.ListTransactions(ctx, riderID, entityTypeRider, page, limit)
	if err != nil {
		return dtos.TransactionsResponse{}, err
	}

	return dtos.TransactionsResponse{
		Transactions: toTransactionDTOs(txs),
		Total:        total,
		Page:         page,
		Limit:        limit,
	}, nil
}

func getZone(balance, withdrawalMinBs float64) models.Zone {
	if withdrawalMinBs > 0 && balance >= withdrawalMinBs {
		return models.ZoneWithdrawalAvailable
	}
	return models.ZoneNormal
}

// RiderEarning returns the rider's net earning from a delivery fee (80%).
func RiderEarning(deliveryFee float64) float64 {
	return deliveryFee * models.RiderShare
}

func toTransactionDTOs(txs []models.Transaction) []dtos.TransactionDTO {
	result := make([]dtos.TransactionDTO, 0, len(txs))
	for _, t := range txs {
		dto := dtos.TransactionDTO{
			ID:            t.ID.String(),
			Amount:        t.Amount,
			BcvRate:       t.BcvRate,
			MovementType:  t.MovementType,
			PaymentMethod: t.PaymentMethod,
			DistanceKm:    t.DistanceKm,
			CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		}
		if t.OrderID != nil {
			s := t.OrderID.String()
			dto.OrderID = &s
		}
		result = append(result, dto)
	}
	return result
}
