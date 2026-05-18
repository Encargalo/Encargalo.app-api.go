package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"github.com/robfig/cron/v3"
)

var venezuelaLoc *time.Location

func init() {
	var err error
	venezuelaLoc, err = time.LoadLocation("America/Caracas")
	if err != nil {
		// UTC-4 fixed fallback if timezone data is unavailable
		venezuelaLoc = time.FixedZone("VET", -4*60*60)
	}
}

type bcvApp struct {
	scraper         ports.BcvScraper
	calendarScraper ports.BcvCalendarScraper
	cache           ports.BcvCache
	holidayRepo     ports.BcvHolidayRepo
}

func New(scraper ports.BcvScraper, calendarScraper ports.BcvCalendarScraper, cache ports.BcvCache, holidayRepo ports.BcvHolidayRepo) ports.BcvApp {
	return &bcvApp{
		scraper:         scraper,
		calendarScraper: calendarScraper,
		cache:           cache,
		holidayRepo:     holidayRepo,
	}
}

func (a *bcvApp) Start(ctx context.Context) {
	go a.fetchOnStartup(ctx)
	go a.syncHolidays(ctx)

	c := cron.New(cron.WithLocation(venezuelaLoc))
	if _, err := c.AddFunc("10 9 * * 1-5", func() { a.runDailyCron(ctx) }); err != nil {
		slog.Error("bcv: failed to register cron", "err", err)
		return
	}
	c.Start()
}

func (a *bcvApp) GetRate(ctx context.Context) (*models.ExchangeRate, error) {
	return a.cache.GetLatest(ctx)
}

// fetchOnStartup does a single scrape to ensure the cache is always populated.
func (a *bcvApp) fetchOnStartup(ctx context.Context) {
	rate, err := a.scraper.Scrape(ctx)
	if err != nil {
		slog.Error("bcv: startup scrape failed", "err", err)
		return
	}
	if err := a.cache.Save(ctx, rate); err != nil {
		slog.Error("bcv: startup cache save failed", "err", err)
		return
	}
	slog.Info("bcv: rate loaded on startup", "rate", rate.Rate, "fecha_valor", rate.Date.Format("2006-01-02"))
}

// syncHolidays scrapes the BCV banking calendar and persists all holidays for
// the current year into postgres.
func (a *bcvApp) syncHolidays(ctx context.Context) {
	year := time.Now().Year()
	dates, err := a.calendarScraper.ScrapeHolidays(ctx)
	if err != nil {
		slog.Error("bcv: failed to scrape banking calendar", "err", err)
		return
	}
	if err := a.holidayRepo.SyncYear(ctx, year, dates); err != nil {
		slog.Error("bcv: failed to sync holidays to db", "err", err)
		return
	}
	slog.Info("bcv: banking holidays synced", "year", year, "count", len(dates))
}

// runDailyCron checks if today is a banking holiday before polling.
// If it is, skips entirely. Otherwise retries every 5 minutes until BCV
// publishes a rate newer than the one in cache.
func (a *bcvApp) runDailyCron(ctx context.Context) {
	today := time.Now().In(venezuelaLoc)

	isHoliday, err := a.holidayRepo.IsHoliday(ctx, today)
	if err != nil {
		slog.Error("bcv: failed to check banking holiday", "err", err)
	}
	if isHoliday {
		slog.Info("bcv: today is a banking holiday, skipping", "date", today.Format("2006-01-02"))
		return
	}

	slog.Info("bcv: daily cron started")
	for {
		updated, err := a.trySyncUpdatedRate(ctx)
		if err != nil {
			slog.Error("bcv: cron sync error", "err", err)
			return
		}
		if updated {
			return
		}
		slog.Info("bcv: rate not yet updated, retrying in 5 minutes")
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
		}
	}
}

// trySyncUpdatedRate returns true if BCV published a newer date than what is cached.
func (a *bcvApp) trySyncUpdatedRate(ctx context.Context) (bool, error) {
	cached, err := a.cache.GetLatest(ctx)
	if err != nil {
		return false, err
	}

	rate, err := a.scraper.Scrape(ctx)
	if err != nil {
		return false, err
	}

	cachedDay := cached.Date.In(venezuelaLoc).Truncate(24 * time.Hour)
	scrapedDay := rate.Date.In(venezuelaLoc).Truncate(24 * time.Hour)

	if scrapedDay.After(cachedDay) {
		if err := a.cache.Save(ctx, rate); err != nil {
			return false, err
		}
		slog.Info("bcv: rate updated", "rate", rate.Rate, "new_fecha_valor", scrapedDay.Format("2006-01-02"))
		return true, nil
	}

	return false, nil
}
