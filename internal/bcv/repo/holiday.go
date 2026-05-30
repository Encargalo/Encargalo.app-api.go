package repo

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"github.com/uptrace/bun"
)

type holidayRepo struct {
	db *bun.DB
}

func NewHolidayRepo(db *bun.DB) ports.BcvHolidayRepo {
	return &holidayRepo{db: db}
}

func (r *holidayRepo) IsHoliday(ctx context.Context, date time.Time) (bool, error) {
	d := date.UTC().Truncate(24 * time.Hour)
	return r.db.NewSelect().
		TableExpr("bcv.banking_holidays").
		Where("date = ?", d.Format("2006-01-02")).
		Exists(ctx)
}

func (r *holidayRepo) SyncYear(ctx context.Context, year int, dates []time.Time) error {
	if len(dates) == 0 {
		return nil
	}

	_, err := r.db.NewDelete().
		TableExpr("bcv.banking_holidays").
		Where("EXTRACT(YEAR FROM date) = ?", year).
		Exec(ctx)
	if err != nil {
		return err
	}

	holidays := make([]*models.BankingHoliday, len(dates))
	for i, d := range dates {
		holidays[i] = &models.BankingHoliday{Date: d.UTC().Truncate(24 * time.Hour)}
	}

	_, err = r.db.NewInsert().Model(&holidays).On("CONFLICT (date) DO NOTHING").Exec(ctx)
	return err
}
