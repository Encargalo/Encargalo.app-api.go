package app

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/models"
	"Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type shopFinanceApp struct {
	repo ports.ShopFinanceRepo
}

func NewShopFinanceApp(repo ports.ShopFinanceRepo) ports.ShopFinanceApp {
	return &shopFinanceApp{repo}
}

func (f *shopFinanceApp) RecordSaleCommission(ctx context.Context, shopID uuid.UUID, orderID uuid.UUID, totalPrice, deliveryFee float64, method string) error {
	base := totalPrice - deliveryFee
	delta := base * models.ShopShare

	tx := models.ShopTransaction{
		ID:            uuid.New(),
		OrderID:       &orderID,
		MovementType:  string(models.MovementTypeSaleDeliveryBank),
		PaymentMethod: &method,
		CreatedAt:     time.Now(),
	}

	_, err := f.repo.RecordMovement(ctx, shopID, models.EntityTypeShop, delta, 0, tx)
	return err
}

func (f *shopFinanceApp) GetBalance(ctx context.Context, shopID uuid.UUID) (dtos.ShopBalanceResponse, error) {
	balance, err := f.repo.GetBalance(ctx, shopID, models.EntityTypeShop)
	if err != nil {
		return dtos.ShopBalanceResponse{}, err
	}

	txs, _, err := f.repo.ListTransactions(ctx, shopID, models.EntityTypeShop, 1, 10)
	if err != nil {
		return dtos.ShopBalanceResponse{}, err
	}

	return dtos.ShopBalanceResponse{
		Balance:      balance,
		Transactions: toShopTransactionDTOs(txs),
	}, nil
}

func (f *shopFinanceApp) ListTransactions(ctx context.Context, shopID uuid.UUID, page, limit int) (dtos.ShopTransactionsResponse, error) {
	if limit > 50 {
		limit = 50
	}

	txs, total, err := f.repo.ListTransactions(ctx, shopID, models.EntityTypeShop, page, limit)
	if err != nil {
		return dtos.ShopTransactionsResponse{}, err
	}

	return dtos.ShopTransactionsResponse{
		Transactions: toShopTransactionDTOs(txs),
		Total:        total,
		Page:         page,
		Limit:        limit,
	}, nil
}

func toShopTransactionDTOs(txs []models.ShopTransaction) []dtos.ShopTransactionDTO {
	result := make([]dtos.ShopTransactionDTO, 0, len(txs))
	for _, t := range txs {
		dto := dtos.ShopTransactionDTO{
			ID:            t.ID.String(),
			Amount:        t.Amount,
			MovementType:  t.MovementType,
			PaymentMethod: t.PaymentMethod,
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
