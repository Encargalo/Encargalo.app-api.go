package ports

import (
	"context"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/models"
	"github.com/google/uuid"
)

type ShopFinanceApp interface {
	RecordSaleCommission(ctx context.Context, shopID uuid.UUID, orderID uuid.UUID, totalPrice, deliveryFee float64, method string, isLaunchPromo, isZeroCommission bool, promoSubsidyAmount float64) error
	GetBalance(ctx context.Context, shopID uuid.UUID) (dtos.ShopBalanceResponse, error)
	ListTransactions(ctx context.Context, shopID uuid.UUID, page, limit int) (dtos.ShopTransactionsResponse, error)
}

type ShopFinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (float64, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, tx models.ShopTransaction) (float64, error)
	ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.ShopTransaction, int, error)
}
