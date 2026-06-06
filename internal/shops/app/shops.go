package app

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	portsShops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type shopsApp struct {
	repo portsShops.ShopsRepo
}

func NewShopsApp(repo portsShops.ShopsRepo) portsShops.ShopsApp {
	return &shopsApp{repo}
}

func (s *shopsApp) GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error) {
	shops, err := s.repo.GetAllShops(ctx, coords)
	if err != nil {
		return nil, err
	}
	return shops, nil
}

func (p *shopsApp) GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error) {
	return p.repo.GetShopsBy(ctx, criteria, coords)
}

func (s *shopsApp) GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error) {
	return s.repo.GetShopByID(ctx, shopID)
}

func (s *shopsApp) SearchShops(ctx context.Context, query dtos.SearchShopsQuery) (dtos.SearchShopsResult, error) {
	candidates, err := s.repo.SearchShopCandidates(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return dtos.SearchShopsResult{}, nil
	}

	shopIDs := make([]uuid.UUID, len(candidates))
	for i, c := range candidates {
		shopIDs[i] = c.ID
	}

	matchedItems, err := s.repo.FindMatchedProducts(ctx, shopIDs, query.Q)
	if err != nil {
		slog.Error("error al cargar productos para búsqueda, se devuelven tiendas sin productos", "error", err)
		matchedItems = make(map[uuid.UUID][]dtos.MatchedProductItem)
	}

	result := make(dtos.SearchShopsResult, 0, len(candidates))
	for _, c := range candidates {
		resp := dtos.SearchShopResponse{ShopResponse: c.ShopResponse}
		resp.MatchedProducts = matchedItems[c.ID]
		result = append(result, resp)
	}

	return result, nil
}
