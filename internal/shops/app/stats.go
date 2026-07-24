package app

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type shopStatsApp struct {
	repo ports.ShopStatsRepo
}

func NewShopStatsApp(repo ports.ShopStatsRepo) ports.ShopStatsApp {
	return &shopStatsApp{repo}
}

func (a *shopStatsApp) GetSummary(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod) (dtos.StatsSummaryResponse, error) {
	curFrom, curTo := currentPeriodRange(period)
	prevFrom, prevTo := previousPeriodRange(period)

	current, err := a.repo.GetPeriodMetrics(ctx, shopID, curFrom, curTo)
	if err != nil {
		return dtos.StatsSummaryResponse{}, err
	}
	current.RejectionRate = rejectionRate(current.OrdersCompleted, current.OrdersRejected)

	previous, err := a.repo.GetPeriodMetrics(ctx, shopID, prevFrom, prevTo)
	if err != nil {
		return dtos.StatsSummaryResponse{}, err
	}
	previous.RejectionRate = rejectionRate(previous.OrdersCompleted, previous.OrdersRejected)

	balance, err := a.repo.GetBalance(ctx, shopID)
	if err != nil {
		return dtos.StatsSummaryResponse{}, err
	}

	return dtos.StatsSummaryResponse{
		Period:         period,
		Current:        current,
		Previous:       previous,
		CurrentBalance: balance,
	}, nil
}

func (a *shopStatsApp) GetTopProducts(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod, limit int) (dtos.TopProductsResponse, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	curFrom, curTo := currentPeriodRange(period)
	prevFrom, prevTo := previousPeriodRange(period)

	current, err := a.repo.GetTopProducts(ctx, shopID, curFrom, curTo, limit)
	if err != nil {
		return dtos.TopProductsResponse{}, err
	}

	previous, err := a.repo.GetTopProducts(ctx, shopID, prevFrom, prevTo, limit)
	if err != nil {
		return dtos.TopProductsResponse{}, err
	}

	return dtos.TopProductsResponse{
		Period:   period,
		Current:  current,
		Previous: previous,
	}, nil
}

func (a *shopStatsApp) GetSalesOverTime(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod, cursor *time.Time) (dtos.SalesOverTimeResponse, error) {
	granularity := granularityFor(period)
	curFrom, curTo := currentPeriodRange(period)
	prevFrom, prevTo := previousPeriodRange(period)

	if cursor != nil {
		curFrom = *cursor
	}

	current, err := a.repo.GetSalesOverTime(ctx, shopID, curFrom, curTo, granularity)
	if err != nil {
		return dtos.SalesOverTimeResponse{}, err
	}

	previous, err := a.repo.GetSalesOverTime(ctx, shopID, prevFrom, prevTo, granularity)
	if err != nil {
		return dtos.SalesOverTimeResponse{}, err
	}

	return dtos.SalesOverTimeResponse{
		Period:      period,
		Granularity: granularity,
		NextCursor:  nil,
		Current:     current,
		Previous:    previous,
	}, nil
}

func (a *shopStatsApp) GetCustomerStats(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod) (dtos.CustomerStatsResponse, error) {
	curFrom, curTo := currentPeriodRange(period)
	prevFrom, prevTo := previousPeriodRange(period)

	current, err := a.repo.GetCustomerMetrics(ctx, shopID, curFrom, curTo)
	if err != nil {
		return dtos.CustomerStatsResponse{}, err
	}

	previous, err := a.repo.GetCustomerMetrics(ctx, shopID, prevFrom, prevTo)
	if err != nil {
		return dtos.CustomerStatsResponse{}, err
	}

	return dtos.CustomerStatsResponse{
		Period:   period,
		Current:  current,
		Previous: previous,
	}, nil
}

func (a *shopStatsApp) ExportTransactions(ctx context.Context, shopID uuid.UUID, period dtos.StatsPeriod) ([]dtos.ShopTransactionDTO, error) {
	from, to := currentPeriodRange(period)
	return a.repo.GetAllTransactionsInPeriod(ctx, shopID, from, to)
}

func currentPeriodRange(period dtos.StatsPeriod) (from, to time.Time) {
	now := time.Now()
	switch period {
	case dtos.PeriodDay:
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 0, 1)
	case dtos.PeriodWeek:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 0, 7)
	default: // month
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, 0)
	}
	return
}

func previousPeriodRange(period dtos.StatsPeriod) (from, to time.Time) {
	now := time.Now()
	switch period {
	case dtos.PeriodDay:
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		from = today.AddDate(0, 0, -1)
		to = today
	case dtos.PeriodWeek:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		thisWeekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		from = thisWeekStart.AddDate(0, 0, -7)
		to = thisWeekStart
	default: // month
		thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		from = thisMonthStart.AddDate(0, -1, 0)
		to = thisMonthStart
	}
	return
}

func rejectionRate(completed, rejected int) float64 {
	total := completed + rejected
	if total == 0 {
		return 0
	}
	return float64(rejected) / float64(total)
}

func granularityFor(period dtos.StatsPeriod) string {
	if period == dtos.PeriodDay {
		return "hour"
	}
	return "day"
}
