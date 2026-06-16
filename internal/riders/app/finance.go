package app

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"github.com/google/uuid"
)

const entityTypeRider = "rider"

type financeApp struct {
	repo ports.FinanceRepo
}

func NewFinanceApp(repo ports.FinanceRepo) ports.FinanceApp {
	return &financeApp{repo}
}

// RecordRideEarning adjusts the rider's balance when an order is completed.
// Cash orders: balance decreases (rider holds the cash).
// Bank orders: balance increases (platform holds the cash and owes the rider).
func (f *financeApp) RecordRideEarning(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, totalPrice float64, method string) error {
	var delta float64
	var movType string

	if isCashPayment(method) {
		delta = -totalPrice
		movType = string(models.MovementTypeRideCash)
	} else {
		delta = deliveryFee * models.RiderShare
		movType = string(models.MovementTypeRideBank)
	}

	tx := models.Transaction{
		ID:            uuid.New(),
		OrderID:       &orderID,
		MovementType:  movType,
		PaymentMethod: &method,
		CreatedAt:     time.Now(),
	}

	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, delta, 0, tx)
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

	return dtos.BalanceResponse{
		Balance:      balance,
		Zone:         getZone(balance),
		Transactions: toTransactionDTOs(txs),
	}, nil
}

func (f *financeApp) RequestWithdrawal(ctx context.Context, riderID uuid.UUID) (dtos.WithdrawalResponse, error) {
	balance, err := f.repo.GetBalance(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.WithdrawalResponse{}, err
	}

	tx := models.Transaction{
		ID:           uuid.New(),
		MovementType: string(models.MovementTypeWithdraw),
		CreatedAt:    time.Now(),
	}

	if _, err = f.repo.RecordMovement(ctx, riderID, entityTypeRider, -balance, models.WithdrawalMinBalance, tx); err != nil {
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

func getZone(balance float64) models.Zone {
	if balance <= models.RiskThreshold {
		return models.ZoneRisk
	}
	if balance >= models.WithdrawalMinBalance {
		return models.ZoneWithdrawalAvailable
	}
	return models.ZoneNormal
}

func isCashPayment(method string) bool {
	return method == "EfectivoBs" || method == "EfectivoDolar"
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
