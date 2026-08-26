package app_test

import (
	"context"
	"testing"
	"time"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// anyTime matchea los argumentos from/to que currentPeriodRange/previousPeriodRange
// calculan a partir de time.Now(): no son predecibles exactamente en el test.
var anyTime = mock.AnythingOfType("time.Time")

func TestGetSummary_ComputesRejectionRateAndIncludesBalance(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetPeriodMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.PeriodMetrics{OrdersCompleted: 3, OrdersRejected: 1}, nil).Once()
	repo.On("GetPeriodMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.PeriodMetrics{OrdersCompleted: 8, OrdersRejected: 0}, nil).Once()
	repo.On("GetBalance", context.Background(), shopID).Return(120.5, nil)

	resp, err := statsApp.GetSummary(context.Background(), shopID, dtos.PeriodDay)

	require.NoError(t, err)
	require.Equal(t, 0.25, resp.Current.RejectionRate)
	require.Equal(t, 0.0, resp.Previous.RejectionRate)
	require.Equal(t, 120.5, resp.CurrentBalance)
}

func TestGetSummary_CurrentMetricsRepoError(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetPeriodMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.PeriodMetrics{}, errcustom.ErrUnexpectedError)

	_, err := statsApp.GetSummary(context.Background(), shopID, dtos.PeriodMonth)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	repo.AssertNotCalled(t, "GetBalance", mock.Anything, mock.Anything)
}

func TestGetSummary_BalanceRepoError(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetPeriodMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.PeriodMetrics{}, nil)
	repo.On("GetBalance", context.Background(), shopID).Return(0.0, errcustom.ErrUnexpectedError)

	_, err := statsApp.GetSummary(context.Background(), shopID, dtos.PeriodWeek)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestGetTopProducts_ZeroLimitDefaultsTo10(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetTopProducts", context.Background(), shopID, anyTime, anyTime, 10).
		Return([]dtos.TopProductItem{}, nil).Twice()

	_, err := statsApp.GetTopProducts(context.Background(), shopID, dtos.PeriodDay, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetTopProducts_OverMaxLimitDefaultsTo10(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetTopProducts", context.Background(), shopID, anyTime, anyTime, 10).
		Return([]dtos.TopProductItem{}, nil).Twice()

	_, err := statsApp.GetTopProducts(context.Background(), shopID, dtos.PeriodDay, 50)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetTopProducts_RepoErrorOnPrevious(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetTopProducts", context.Background(), shopID, anyTime, anyTime, 5).
		Return([]dtos.TopProductItem{}, nil).Once()
	repo.On("GetTopProducts", context.Background(), shopID, anyTime, anyTime, 5).
		Return(nil, errcustom.ErrUnexpectedError).Once()

	_, err := statsApp.GetTopProducts(context.Background(), shopID, dtos.PeriodWeek, 5)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

// GetSalesOverTime usa granularity "hour" para el periodo día y "day" para
// semana/mes (granularityFor), y el cursor reemplaza el from calculado por
// currentPeriodRange cuando viene provisto.
func TestGetSalesOverTime_DayUsesHourGranularity(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetSalesOverTime", context.Background(), shopID, anyTime, anyTime, "hour").
		Return([]dtos.SalesDataPoint{}, nil).Twice()

	resp, err := statsApp.GetSalesOverTime(context.Background(), shopID, dtos.PeriodDay, nil)

	require.NoError(t, err)
	require.Equal(t, "hour", resp.Granularity)
}

func TestGetSalesOverTime_WeekUsesDayGranularity(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetSalesOverTime", context.Background(), shopID, anyTime, anyTime, "day").
		Return([]dtos.SalesDataPoint{}, nil).Twice()

	resp, err := statsApp.GetSalesOverTime(context.Background(), shopID, dtos.PeriodWeek, nil)

	require.NoError(t, err)
	require.Equal(t, "day", resp.Granularity)
}

func TestGetSalesOverTime_CursorOverridesCurrentFrom(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	cursor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo.On("GetSalesOverTime", context.Background(), shopID, cursor, anyTime, "day").
		Return([]dtos.SalesDataPoint{}, nil).Once()
	repo.On("GetSalesOverTime", context.Background(), shopID, anyTime, anyTime, "day").
		Return([]dtos.SalesDataPoint{}, nil).Once()

	_, err := statsApp.GetSalesOverTime(context.Background(), shopID, dtos.PeriodMonth, &cursor)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetSalesOverTime_RepoError(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetSalesOverTime", context.Background(), shopID, anyTime, anyTime, "day").
		Return(nil, errcustom.ErrUnexpectedError)

	_, err := statsApp.GetSalesOverTime(context.Background(), shopID, dtos.PeriodMonth, nil)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestGetCustomerStats_Success(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetCustomerMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.CustomerMetrics{NewCustomers: 2, ReturningCustomers: 5}, nil).Once()
	repo.On("GetCustomerMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.CustomerMetrics{NewCustomers: 1, ReturningCustomers: 3}, nil).Once()

	resp, err := statsApp.GetCustomerStats(context.Background(), shopID, dtos.PeriodDay)

	require.NoError(t, err)
	require.Equal(t, 2, resp.Current.NewCustomers)
	require.Equal(t, 1, resp.Previous.NewCustomers)
}

func TestGetCustomerStats_RepoError(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetCustomerMetrics", context.Background(), shopID, anyTime, anyTime).
		Return(dtos.CustomerMetrics{}, errcustom.ErrUnexpectedError)

	_, err := statsApp.GetCustomerStats(context.Background(), shopID, dtos.PeriodDay)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestExportTransactions_DelegatesToRepo(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetAllTransactionsInPeriod", context.Background(), shopID, anyTime, anyTime).
		Return([]dtos.ShopTransactionDTO{{ID: "tx1"}}, nil)

	resp, err := statsApp.ExportTransactions(context.Background(), shopID, dtos.PeriodMonth)

	require.NoError(t, err)
	require.Len(t, resp, 1)
}

func TestExportTransactions_RepoError(t *testing.T) {
	repo := mocksshops.NewShopStatsRepo(t)
	statsApp := app.NewShopStatsApp(repo)

	shopID := uuid.New()
	repo.On("GetAllTransactionsInPeriod", context.Background(), shopID, anyTime, anyTime).
		Return(nil, errcustom.ErrUnexpectedError)

	_, err := statsApp.ExportTransactions(context.Background(), shopID, dtos.PeriodMonth)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
