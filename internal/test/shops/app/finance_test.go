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

func TestRecordSaleCommission_RegularSale(t *testing.T) {
	repo := new(mocksshops.ShopFinanceRepo)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	orderID := uuid.New()

	// totalPrice=52 incluye delivery=2 -> subtotal=50, comision estandar
	// del negocio es 96% -> delta esperado 48.
	repo.On("RecordMovement", context.Background(), shopID, models.EntityTypeShop, 48.0, 0.0, mock.AnythingOfType("models.ShopTransaction")).
		Return(48.0, nil)

	err := financeApp.RecordSaleCommission(context.Background(), shopID, orderID, 52, 2, "cash", false, false, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "RecordMovement", context.Background(), shopID, models.EntityTypePlatformSubsidy, mock.Anything, mock.Anything, mock.Anything)
}

func TestRecordSaleCommission_ZeroCommission(t *testing.T) {
	repo := new(mocksshops.ShopFinanceRepo)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	orderID := uuid.New()

	// Dentro de los primeros 8 días el negocio recibe el 100% del subtotal.
	repo.On("RecordMovement", context.Background(), shopID, models.EntityTypeShop, 50.0, 0.0, mock.AnythingOfType("models.ShopTransaction")).
		Return(50.0, nil)

	err := financeApp.RecordSaleCommission(context.Background(), shopID, orderID, 52, 2, "cash", false, true, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRecordSaleCommission_LaunchPromoSubsidy(t *testing.T) {
	repo := new(mocksshops.ShopFinanceRepo)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	orderID := uuid.New()

	// Cliente pagó totalPrice=27 (delivery=2, subtotal pagado=25) porque tuvo
	// 50% de descuento; promoSubsidyAmount=25 reconstruye el subtotal
	// original de 50, sobre el cual el negocio recibe el 100% (día 1).
	repo.On("RecordMovement", context.Background(), shopID, models.EntityTypeShop, 50.0, 0.0, mock.AnythingOfType("models.ShopTransaction")).
		Return(50.0, nil)
	repo.On("RecordMovement", context.Background(), shopID, models.EntityTypePlatformSubsidy, -25.0, 0.0, mock.AnythingOfType("models.ShopTransaction")).
		Return(-25.0, nil)

	err := financeApp.RecordSaleCommission(context.Background(), shopID, orderID, 27, 2, "cash", true, true, 25)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}
