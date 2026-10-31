package repo

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type scheduleRow struct {
	ShopID  uuid.UUID `bun:"shop_id"`
	Day     int16     `bun:"day"`
	Open    string    `bun:"open"`
	Close   string    `bun:"close"`
	Enabled bool      `bun:"enabled"`
}

func getShopSchedule(ctx context.Context, db bun.IDB, shopID uuid.UUID) ([]dtos.ScheduleResponse, error) {
	byShop, err := getSchedulesByShopIDs(ctx, db, []uuid.UUID{shopID})
	if err != nil {
		return nil, err
	}
	return byShop[shopID], nil
}

func getSchedulesByShopIDs(ctx context.Context, db bun.IDB, shopIDs []uuid.UUID) (map[uuid.UUID][]dtos.ScheduleResponse, error) {
	result := make(map[uuid.UUID][]dtos.ScheduleResponse, len(shopIDs))
	if len(shopIDs) == 0 {
		return result, nil
	}

	var rows []scheduleRow
	err := db.NewSelect().
		TableExpr("business.schedules AS sc").
		Join("JOIN business.week AS w ON w.id = sc.day_id").
		ColumnExpr("sc.shop_id AS shop_id, w.day AS day, sc.open AS open, sc.close AS close, sc.enabled AS enabled").
		Where("sc.shop_id IN (?)", bun.In(shopIDs)).
		Order("w.day ASC").
		Scan(ctx, &rows)
	if err != nil {
		slog.Error("error al obtener horario de locales", "error", err, "shopIDs", shopIDs)
		return nil, errcustom.Unexpected(err)
	}

	for _, r := range rows {
		result[r.ShopID] = append(result[r.ShopID], dtos.ScheduleResponse{
			DayOfWeek: r.Day,
			Open:      r.Open,
			Close:     r.Close,
			Enabled:   r.Enabled,
		})
	}
	return result, nil
}

func replaceShopSchedule(ctx context.Context, db *bun.DB, shopID uuid.UUID, schedules []dtos.ScheduleInput) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var weekRows []struct {
			ID  uuid.UUID `bun:"id"`
			Day int16     `bun:"day"`
		}
		if err := tx.NewSelect().TableExpr("business.week").Column("id", "day").Scan(ctx, &weekRows); err != nil {
			slog.Error("error al cargar business.week", "error", err)
			return errcustom.Unexpected(err)
		}
		dayID := make(map[int16]uuid.UUID, len(weekRows))
		for _, w := range weekRows {
			dayID[w.Day] = w.ID
		}

		if _, err := tx.NewDelete().TableExpr("business.schedules").Where("shop_id = ?", shopID).Exec(ctx); err != nil {
			slog.Error("error al borrar horario previo del shop", "error", err, "shopID", shopID)
			return errcustom.Unexpected(err)
		}

		if len(schedules) == 0 {
			return nil
		}

		rows := make([]shopsmodels.Schedule, 0, len(schedules))
		for _, s := range schedules {
			did, ok := dayID[s.DayOfWeek]
			if !ok {
				return errcustom.Unexpected(nil)
			}
			rows = append(rows, shopsmodels.Schedule{
				ShopID:  shopID,
				DayID:   did,
				Open:    s.Open,
				Close:   s.Close,
				Enabled: s.EnabledOrDefault(),
			})
		}

		if _, err := tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
			slog.Error("error al insertar horario del shop", "error", err, "shopID", shopID)
			return errcustom.Unexpected(err)
		}
		return nil
	})
}
