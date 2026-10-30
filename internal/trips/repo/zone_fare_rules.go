package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// activeZoneFareCellsCacheKey cachea el mapa completo celda→regla activa en
// una sola clave: tripsApp.EstimateFare lo consulta en cada cotización, y el
// universo de celdas con regla es chico (decenas por zona, pocas zonas).
const (
	activeZoneFareCellsCacheKey = "trips:zone_fare_rules:active_cells"
	activeZoneFareCellsCacheTTL = time.Hour
)

type zoneFareRulesRepo struct {
	db    *bun.DB
	cache cache.Cache
}

func NewZoneFareRulesRepo(db *bun.DB, cache cache.Cache) ports.ZoneFareRulesRepo {
	return &zoneFareRulesRepo{db: db, cache: cache}
}

func (r *zoneFareRulesRepo) invalidateActiveCellsCache(ctx context.Context) {
	if err := r.cache.Clear(ctx, activeZoneFareCellsCacheKey); err != nil {
		slog.Error("error al limpiar el cache de celdas de zona activas", "error", err)
	}
}

func (r *zoneFareRulesRepo) Create(ctx context.Context, rule *models.ZoneFareRule, cells []string) error {
	rule.ZoneID = id.Ensure(rule.ZoneID)
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(rule).Exec(ctx); err != nil {
			return err
		}
		return insertZoneFareCells(ctx, tx, rule.ZoneID, cells)
	})
	if err != nil {
		slog.Error("zoneFareRules Create failed", "zoneID", rule.ZoneID, "error", err)
		return errcustom.Unexpected(err)
	}
	r.invalidateActiveCellsCache(ctx)
	return nil
}

func (r *zoneFareRulesRepo) Update(ctx context.Context, rule *models.ZoneFareRule, cells []string) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model(rule).
			Column("name", "included_km",
				"day_min_fare_usd", "day_extra_rate_per_km_usd",
				"night_min_fare_usd", "night_extra_rate_per_km_usd",
				"early_morning_min_fare_usd", "early_morning_extra_rate_per_km_usd",
				"updated_at").
			WherePK().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model((*models.ZoneFareCell)(nil)).Where("zone_id = ?", rule.ZoneID).Exec(ctx); err != nil {
			return err
		}
		return insertZoneFareCells(ctx, tx, rule.ZoneID, cells)
	})
	if err != nil {
		slog.Error("zoneFareRules Update failed", "zoneID", rule.ZoneID, "error", err)
		return errcustom.Unexpected(err)
	}
	r.invalidateActiveCellsCache(ctx)
	return nil
}

func insertZoneFareCells(ctx context.Context, tx bun.Tx, zoneID uuid.UUID, cells []string) error {
	rows := make([]*models.ZoneFareCell, len(cells))
	for i, cell := range cells {
		rows[i] = &models.ZoneFareCell{H3Cell: cell, ZoneID: zoneID}
	}
	_, err := tx.NewInsert().Model(&rows).Exec(ctx)
	return err
}

func (r *zoneFareRulesRepo) GetByID(ctx context.Context, zoneID uuid.UUID) (*models.ZoneFareRule, error) {
	var rule models.ZoneFareRule
	err := r.db.NewSelect().Model(&rule).Where("zone_id = ?", zoneID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("zoneFareRules GetByID failed", "zoneID", zoneID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return &rule, nil
}

func (r *zoneFareRulesRepo) GetCellsByZoneID(ctx context.Context, zoneID uuid.UUID) ([]string, error) {
	var cells []models.ZoneFareCell
	if err := r.db.NewSelect().Model(&cells).Where("zone_id = ?", zoneID).Scan(ctx); err != nil {
		slog.Error("zoneFareRules GetCellsByZoneID failed", "zoneID", zoneID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = c.H3Cell
	}
	return out, nil
}

func (r *zoneFareRulesRepo) List(ctx context.Context) ([]models.ZoneFareRule, error) {
	var rules []models.ZoneFareRule
	if err := r.db.NewSelect().Model(&rules).Order("created_at ASC").Scan(ctx); err != nil {
		slog.Error("zoneFareRules List failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return rules, nil
}

func (r *zoneFareRulesRepo) Deactivate(ctx context.Context, zoneID uuid.UUID) error {
	res, err := r.db.NewUpdate().
		Model((*models.ZoneFareRule)(nil)).
		Set("active = FALSE, updated_at = now()").
		Where("zone_id = ?", zoneID).
		Exec(ctx)
	if err != nil {
		slog.Error("zoneFareRules Deactivate failed", "zoneID", zoneID, "error", err)
		return errcustom.Unexpected(err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errcustom.ErrNotFound
	}
	r.invalidateActiveCellsCache(ctx)
	return nil
}

// activeZoneFareCellRow es el join plano celda→regla activa, para no acoplar
// el cache a la forma de dos tablas ni a los tags bun de models.ZoneFareRule.
type activeZoneFareCellRow struct {
	H3Cell                        string    `json:"h3_cell" bun:"h3_cell"`
	ZoneID                        uuid.UUID `json:"zone_id" bun:"zone_id"`
	Name                          string    `json:"name" bun:"name"`
	IncludedKm                    float64   `json:"included_km" bun:"included_km"`
	DayMinFareUSD                 float64   `json:"day_min_fare_usd" bun:"day_min_fare_usd"`
	DayExtraRatePerKmUSD          float64   `json:"day_extra_rate_per_km_usd" bun:"day_extra_rate_per_km_usd"`
	NightMinFareUSD               float64   `json:"night_min_fare_usd" bun:"night_min_fare_usd"`
	NightExtraRatePerKmUSD        float64   `json:"night_extra_rate_per_km_usd" bun:"night_extra_rate_per_km_usd"`
	EarlyMorningMinFareUSD        float64   `json:"early_morning_min_fare_usd" bun:"early_morning_min_fare_usd"`
	EarlyMorningExtraRatePerKmUSD float64   `json:"early_morning_extra_rate_per_km_usd" bun:"early_morning_extra_rate_per_km_usd"`
	Active                        bool      `json:"active" bun:"active"`
	CreatedAt                     time.Time `json:"created_at" bun:"created_at"`
	UpdatedAt                     time.Time `json:"updated_at" bun:"updated_at"`
}

func (row activeZoneFareCellRow) toRule() models.ZoneFareRule {
	return models.ZoneFareRule{
		ZoneID:                        row.ZoneID,
		Name:                          row.Name,
		IncludedKm:                    row.IncludedKm,
		DayMinFareUSD:                 row.DayMinFareUSD,
		DayExtraRatePerKmUSD:          row.DayExtraRatePerKmUSD,
		NightMinFareUSD:               row.NightMinFareUSD,
		NightExtraRatePerKmUSD:        row.NightExtraRatePerKmUSD,
		EarlyMorningMinFareUSD:        row.EarlyMorningMinFareUSD,
		EarlyMorningExtraRatePerKmUSD: row.EarlyMorningExtraRatePerKmUSD,
		Active:                        row.Active,
		CreatedAt:                     row.CreatedAt,
		UpdatedAt:                     row.UpdatedAt,
	}
}

func (r *zoneFareRulesRepo) FindActiveByCell(ctx context.Context, h3Cell string) (*models.ZoneFareRule, error) {
	cellMap, err := r.activeCellMap(ctx)
	if err != nil {
		return nil, err
	}
	row, ok := cellMap[h3Cell]
	if !ok {
		return nil, errcustom.ErrNotFound
	}
	rule := row.toRule()
	return &rule, nil
}

func (r *zoneFareRulesRepo) activeCellMap(ctx context.Context) (map[string]activeZoneFareCellRow, error) {
	cached, _ := r.cache.Get(ctx, activeZoneFareCellsCacheKey)
	if cached != "" {
		var cellMap map[string]activeZoneFareCellRow
		if err := json.Unmarshal([]byte(cached), &cellMap); err == nil {
			return cellMap, nil
		}
	}

	var rows []activeZoneFareCellRow
	err := r.db.NewRaw(`
		SELECT c.h3_cell, r.zone_id, r.name, r.included_km,
			r.day_min_fare_usd, r.day_extra_rate_per_km_usd,
			r.night_min_fare_usd, r.night_extra_rate_per_km_usd,
			r.early_morning_min_fare_usd, r.early_morning_extra_rate_per_km_usd,
			r.active, r.created_at, r.updated_at
		FROM trips.zone_fare_cells AS c
		JOIN trips.zone_fare_rules AS r ON r.zone_id = c.zone_id
		WHERE r.active = TRUE
	`).Scan(ctx, &rows)
	if err != nil {
		slog.Error("zoneFareRules activeCellMap failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}

	cellMap := make(map[string]activeZoneFareCellRow, len(rows))
	for _, row := range rows {
		cellMap[row.H3Cell] = row
	}

	if err := r.cache.Set(ctx, activeZoneFareCellsCacheKey, cellMap, activeZoneFareCellsCacheTTL); err != nil {
		slog.Error("error al guardar las celdas de zona activas en el cache", "error", err)
	}

	return cellMap, nil
}
