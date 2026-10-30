package app

import (
	"context"
	"fmt"
	"time"

	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type zoneFareRulesApp struct {
	repo ports.ZoneFareRulesRepo
}

func NewZoneFareRulesApp(repo ports.ZoneFareRulesRepo) ports.ZoneFareRulesApp {
	return &zoneFareRulesApp{repo: repo}
}

func (a *zoneFareRulesApp) CreateZoneFareRule(ctx context.Context, req dtos.ZoneFareRuleRequest) (dtos.ZoneFareRuleResponse, error) {
	cells, err := polygonToZoneCells(req.Polygon)
	if err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}

	now := time.Now()
	rule := &models.ZoneFareRule{
		ZoneID:                        id.New(),
		Name:                          req.Name,
		IncludedKm:                    req.IncludedKm,
		DayMinFareUSD:                 req.DayMinFareUSD,
		DayExtraRatePerKmUSD:          req.DayExtraRatePerKmUSD,
		NightMinFareUSD:               req.NightMinFareUSD,
		NightExtraRatePerKmUSD:        req.NightExtraRatePerKmUSD,
		EarlyMorningMinFareUSD:        req.EarlyMorningMinFareUSD,
		EarlyMorningExtraRatePerKmUSD: req.EarlyMorningExtraRatePerKmUSD,
		Active:                        true,
		CreatedAt:                     now,
		UpdatedAt:                     now,
	}
	if err := a.repo.Create(ctx, rule, cells); err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}
	return toZoneFareRuleResponse(rule, cells), nil
}

func (a *zoneFareRulesApp) UpdateZoneFareRule(ctx context.Context, zoneID uuid.UUID, req dtos.ZoneFareRuleRequest) (dtos.ZoneFareRuleResponse, error) {
	existing, err := a.repo.GetByID(ctx, zoneID)
	if err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}
	cells, err := polygonToZoneCells(req.Polygon)
	if err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}

	existing.Name = req.Name
	existing.IncludedKm = req.IncludedKm
	existing.DayMinFareUSD = req.DayMinFareUSD
	existing.DayExtraRatePerKmUSD = req.DayExtraRatePerKmUSD
	existing.NightMinFareUSD = req.NightMinFareUSD
	existing.NightExtraRatePerKmUSD = req.NightExtraRatePerKmUSD
	existing.EarlyMorningMinFareUSD = req.EarlyMorningMinFareUSD
	existing.EarlyMorningExtraRatePerKmUSD = req.EarlyMorningExtraRatePerKmUSD
	existing.UpdatedAt = time.Now()

	if err := a.repo.Update(ctx, existing, cells); err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}
	return toZoneFareRuleResponse(existing, cells), nil
}

func (a *zoneFareRulesApp) ListZoneFareRules(ctx context.Context) (dtos.ZoneFareRulesResponse, error) {
	rules, err := a.repo.List(ctx)
	if err != nil {
		return dtos.ZoneFareRulesResponse{}, err
	}

	out := make([]dtos.ZoneFareRuleResponse, 0, len(rules))
	for i := range rules {
		cells, err := a.repo.GetCellsByZoneID(ctx, rules[i].ZoneID)
		if err != nil {
			return dtos.ZoneFareRulesResponse{}, err
		}
		out = append(out, toZoneFareRuleResponse(&rules[i], cells))
	}
	return dtos.ZoneFareRulesResponse{Zones: out}, nil
}

func (a *zoneFareRulesApp) GetZoneFareRule(ctx context.Context, zoneID uuid.UUID) (dtos.ZoneFareRuleResponse, error) {
	rule, err := a.repo.GetByID(ctx, zoneID)
	if err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}
	cells, err := a.repo.GetCellsByZoneID(ctx, zoneID)
	if err != nil {
		return dtos.ZoneFareRuleResponse{}, err
	}
	return toZoneFareRuleResponse(rule, cells), nil
}

func (a *zoneFareRulesApp) DeactivateZoneFareRule(ctx context.Context, zoneID uuid.UUID) error {
	return a.repo.Deactivate(ctx, zoneID)
}

// polygonToZoneCells convierte el polígono del request a celdas H3 en la
// misma resolución que trips/app usa para resolver la zona del pickup
// (geo.ZoneAnalyticsResolution) — si se usara otra, FindActiveByCell nunca matchearía.
func polygonToZoneCells(points []dtos.LatLngPoint) ([]string, error) {
	vertices := make([]geo.LatLng, len(points))
	for i, p := range points {
		vertices[i] = geo.LatLng{Lat: p.Lat, Lng: p.Lng}
	}
	cells, err := geo.PolygonToCells(vertices, geo.ZoneAnalyticsResolution)
	if err != nil {
		return nil, fmt.Errorf("%w: polígono inválido: %s", errcustom.ErrInvalidStatus, err.Error())
	}
	if len(cells) == 0 {
		return nil, fmt.Errorf("%w: el polígono no cubre ninguna celda", errcustom.ErrInvalidStatus)
	}
	return cells, nil
}

func toZoneFareRuleResponse(rule *models.ZoneFareRule, cells []string) dtos.ZoneFareRuleResponse {
	return dtos.ZoneFareRuleResponse{
		ZoneID:                        rule.ZoneID,
		Name:                          rule.Name,
		IncludedKm:                    rule.IncludedKm,
		DayMinFareUSD:                 rule.DayMinFareUSD,
		DayExtraRatePerKmUSD:          rule.DayExtraRatePerKmUSD,
		NightMinFareUSD:               rule.NightMinFareUSD,
		NightExtraRatePerKmUSD:        rule.NightExtraRatePerKmUSD,
		EarlyMorningMinFareUSD:        rule.EarlyMorningMinFareUSD,
		EarlyMorningExtraRatePerKmUSD: rule.EarlyMorningExtraRatePerKmUSD,
		H3Cells:                       cells,
		Active:                        rule.Active,
		CreatedAt:                     rule.CreatedAt,
		UpdatedAt:                     rule.UpdatedAt,
	}
}
