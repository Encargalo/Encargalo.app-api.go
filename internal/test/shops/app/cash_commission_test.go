package app_test

import (
	"context"
	"testing"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecordSaleCommission_CashUsesCashMovement(t *testing.T) {
	repo := mocksshops.NewShopFinanceRepo(t)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	orderID := uuid.New()

	repo.On("RecordMovement", mock.Anything, shopID, models.EntityTypeShop, 45.0, 0.0, mock.MatchedBy(func(tx models.ShopTransaction) bool {
		return tx.MovementType == string(models.MovementTypeSaleDeliveryCash) &&
			tx.PaymentMethod != nil && *tx.PaymentMethod == models.MethodEfectivo
	})).Return(45.0, nil)

	err := financeApp.RecordSaleCommission(context.Background(), shopID, orderID, 52, 2, models.MethodEfectivo, false, false, 0)

	require.NoError(t, err)
}
