package app

import (
	"context"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	portsShops "Encargalo.app-api.go/internal/shops/domain/ports/shops"
)

type shopsApp struct {
	repo portsShops.ShopsRepo
}

func NewShopsApp(repo portsShops.ShopsRepo) portsShops.ShopsApp {
	return &shopsApp{repo}
}

func (s *shopsApp) GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error) {
	return s.repo.GetAllShops(ctx, coords)
}

func (p *shopsApp) GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error) {
	return p.repo.GetShopsBy(ctx, criteria, coords)
}
