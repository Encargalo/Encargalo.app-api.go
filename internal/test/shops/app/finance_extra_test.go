package app_test

import (
	"context"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShopFinance_GetBalance_Success(t *testing.T) {
	repo := mocksshops.NewShopFinanceRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	financeApp := app.NewShopFinanceApp(repo, bcv)

	shopID := uuid.New()
	repo.On("GetBalance", context.Background(), shopID, models.EntityTypeShop).
		Return(models.ShopBalance{Amount: 100, Currency: models.CurrencyUSD}, nil)
	repo.On("ListTransactions", context.Background(), shopID, models.EntityTypeShop, 1, 10).
		Return([]models.ShopTransaction{{ID: uuid.New(), Amount: 100, Currency: models.CurrencyUSD, MovementType: "sale"}}, 1, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := financeApp.GetBalance(context.Background(), shopID)

	require.NoError(t, err)
	require.Equal(t, 100.0, resp.BalanceUSD)
	require.Equal(t, 40.0, resp.BcvRate)
	require.Len(t, resp.Transactions, 1)
}

func TestShopFinance_GetBalance_RepoError(t *testing.T) {
	repo := mocksshops.NewShopFinanceRepo(t)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	repo.On("GetBalance", context.Background(), shopID, models.EntityTypeShop).
		Return(models.ShopBalance{}, errcustom.ErrUnexpectedError)

	_, err := financeApp.GetBalance(context.Background(), shopID)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	repo.AssertNotCalled(t, "ListTransactions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// spotBcvRate no debe romper el balance cuando el proveedor BCV falla: los
// montos en Bs quedan en 0 en vez de propagar el error (mismo criterio que
// orderApp.bcvRate).
func TestShopFinance_GetBalance_BcvUnavailableDefaultsRateToZero(t *testing.T) {
	repo := mocksshops.NewShopFinanceRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	financeApp := app.NewShopFinanceApp(repo, bcv)

	shopID := uuid.New()
	repo.On("GetBalance", context.Background(), shopID, models.EntityTypeShop).
		Return(models.ShopBalance{Amount: 100, Currency: models.CurrencyUSD}, nil)
	repo.On("ListTransactions", context.Background(), shopID, models.EntityTypeShop, 1, 10).
		Return([]models.ShopTransaction{}, 0, nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errcustom.ErrUnexpectedError)

	resp, err := financeApp.GetBalance(context.Background(), shopID)

	require.NoError(t, err)
	require.Equal(t, 0.0, resp.BcvRate)
}

func TestShopFinance_ListTransactions_CapsLimitTo50(t *testing.T) {
	repo := mocksshops.NewShopFinanceRepo(t)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	repo.On("ListTransactions", context.Background(), shopID, models.EntityTypeShop, 1, 50).
		Return([]models.ShopTransaction{}, 0, nil)

	resp, err := financeApp.ListTransactions(context.Background(), shopID, 1, 200)

	require.NoError(t, err)
	require.Equal(t, 50, resp.Limit)
	repo.AssertExpectations(t)
}

func TestShopFinance_ListTransactions_RepoError(t *testing.T) {
	repo := mocksshops.NewShopFinanceRepo(t)
	financeApp := app.NewShopFinanceApp(repo, nil)

	shopID := uuid.New()
	repo.On("ListTransactions", context.Background(), shopID, models.EntityTypeShop, 1, 10).
		Return(nil, 0, errcustom.ErrUnexpectedError)

	_, err := financeApp.ListTransactions(context.Background(), shopID, 1, 10)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
