package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/models"
)

type BcvApp interface {
	Start(ctx context.Context)
	GetRate(ctx context.Context) (*models.ExchangeRate, error)
}

type BcvScraper interface {
	Scrape(ctx context.Context) (*models.ExchangeRate, error)
}

type BcvCalendarScraper interface {
	ScrapeHolidays(ctx context.Context) ([]time.Time, error)
}

type BcvCache interface {
	Save(ctx context.Context, rate *models.ExchangeRate) error
	GetLatest(ctx context.Context) (*models.ExchangeRate, error)
}

type BcvHolidayRepo interface {
	IsHoliday(ctx context.Context, date time.Time) (bool, error)
	SyncYear(ctx context.Context, year int, dates []time.Time) error
}
