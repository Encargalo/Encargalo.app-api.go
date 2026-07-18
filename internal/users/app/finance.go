package app

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

type customerFinanceApp struct {
	repo ports.CustomerFinanceRepo
}

func NewCustomerFinanceApp(repo ports.CustomerFinanceRepo) ports.CustomerFinanceApp {
	return &customerFinanceApp{repo}
}

func (f *customerFinanceApp) GetBalance(ctx context.Context, customerID uuid.UUID) (dto.CustomerBalanceResponse, error) {
	balance, err := f.repo.GetBalance(ctx, customerID, models.EntityTypeCustomer)
	if err != nil {
		return dto.CustomerBalanceResponse{}, err
	}

	txs, _, err := f.repo.ListTransactions(ctx, customerID, models.EntityTypeCustomer, 1, 10)
	if err != nil {
		return dto.CustomerBalanceResponse{}, err
	}

	items := make([]dto.CustomerTransactionItem, len(txs))
	for i, t := range txs {
		items[i] = dto.CustomerTransactionItem{
			ID:           t.ID.String(),
			Amount:       t.Amount,
			MovementType: t.MovementType,
			CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		}
		if t.OrderID != nil {
			s := t.OrderID.String()
			items[i].OrderID = &s
		}
	}

	return dto.CustomerBalanceResponse{
		Balance:      balance,
		Transactions: items,
	}, nil
}

func (f *customerFinanceApp) GetBalanceUSD(ctx context.Context, customerID uuid.UUID) (float64, error) {
	return f.repo.GetBalance(ctx, customerID, models.EntityTypeCustomer)
}

func (f *customerFinanceApp) CreditPaymentOverage(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountUSD, 0, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypePaymentOverage),
	})
	return err
}

func (f *customerFinanceApp) CreditPaymentPartial(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountUSD, 0, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypePaymentPartialCredit),
	})
	return err
}

func (f *customerFinanceApp) CreditOrderRejected(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountUSD, 0, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypeOrderRejectedRefund),
	})
	return err
}

func (f *customerFinanceApp) DebitForOrder(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	// minBalance = amountUSD exige saldo >= monto a debitar antes de aplicar delta negativo.
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, -amountUSD, amountUSD, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypeBalancePayment),
	})
	return err
}

func (f *customerFinanceApp) RefundBalancePayment(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountUSD, 0, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypeBalancePaymentRefund),
	})
	return err
}
