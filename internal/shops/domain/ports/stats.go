package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
)

type ShopStatsApp interface {
	GetSummary(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod) (dtos.StatsSummaryResponse, error)
	GetTopProducts(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod, limit int) (dtos.TopProductsResponse, error)
	GetSalesOverTime(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod, cursor *time.Time) (dtos.SalesOverTimeResponse, error)
	GetCustomerStats(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod) (dtos.CustomerStatsResponse, error)
	ExportTransactions(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod) ([]dtos.ShopTransactionDTO, error)
}

type ShopStatsRepo interface {
	GetBalance(ctx context.Context, shopID uuid.UUID) (float64, error)
	GetPeriodMetrics(ctx context.Context, shopID uuid.UUID, from, to time.Time) (dtos.PeriodMetrics, error)
	GetTopProducts(ctx context.Context, shopID uuid.UUID, from, to time.Time, limit int) ([]dtos.TopProductItem, error)
	GetSalesOverTime(ctx context.Context, shopID uuid.UUID, from, to time.Time, granularity string) ([]dtos.SalesDataPoint, error)
	GetCustomerMetrics(ctx context.Context, shopID uuid.UUID, from, to time.Time) (dtos.CustomerMetrics, error)
	GetAllTransactionsInPeriod(ctx context.Context, shopID uuid.UUID, from, to time.Time) ([]dtos.ShopTransactionDTO, error)
}
