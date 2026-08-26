package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/finance/app"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	mockbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockfinance "Encargalo.app-api.go/internal/mocks/finance/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListPayables_ConvertsToUSDWithSpotRate(t *testing.T) {
	repo := mockfinance.NewStaffFinanceRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	svc := app.NewStaffFinanceApp(repo, bcv, nil)

	sur := "Verdi"
	repo.On("ListPayables", mock.Anything, financeModels.EntityTypeShop).Return([]financeModels.PayableRow{
		{EntityID: uuid.New(), Amount: 100, Currency: financeModels.CurrencyUSD, Name: "Pizzería Don Luigi", SurName: &sur, UpdatedAt: time.Now()},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 50}, nil)

	resp, err := svc.ListPayables(context.Background(), financeModels.EntityTypeShop)

	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Equal(t, 100.0, resp.Items[0].BalanceUSD)
	require.Equal(t, "Verdi", resp.Items[0].SurName)
}

func TestListPayables_RepoError(t *testing.T) {
	repo := mockfinance.NewStaffFinanceRepo(t)
	svc := app.NewStaffFinanceApp(repo, nil, nil)

	repo.On("ListPayables", mock.Anything, financeModels.EntityTypeRider).
		Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.ListPayables(context.Background(), financeModels.EntityTypeRider)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

// spotBcvRate (usado por ListPayables) no debe romper el listado cuando el
// bcv no está disponible: BcvRate simplemente queda en 0, mismo criterio
// que shopFinanceApp.spotBcvRate/orderApp.bcvRate.
func TestListPayables_BcvUnavailableDefaultsRateToZero(t *testing.T) {
	repo := mockfinance.NewStaffFinanceRepo(t)
	svc := app.NewStaffFinanceApp(repo, nil, nil)

	repo.On("ListPayables", mock.Anything, financeModels.EntityTypeShop).Return([]financeModels.PayableRow{
		{EntityID: uuid.New(), Amount: 100, Currency: financeModels.CurrencyUSD, UpdatedAt: time.Now()},
	}, nil)

	resp, err := svc.ListPayables(context.Background(), financeModels.EntityTypeShop)

	require.NoError(t, err)
	require.Equal(t, 0.0, resp.Items[0].BcvRate)
}
