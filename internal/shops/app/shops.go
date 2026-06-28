package app

import (
	"context"
	"log/slog"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	portsShops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type shopsApp struct {
	repo        portsShops.ShopsRepo
	broadcaster portsShops.ShopStatusBroadcaster
	bcv         bcvPorts.BcvApp
}

func NewShopsApp(repo portsShops.ShopsRepo, broadcaster portsShops.ShopStatusBroadcaster, bcv bcvPorts.BcvApp) portsShops.ShopsApp {
	return &shopsApp{repo, broadcaster, bcv}
}

func (s *shopsApp) bcvRate(ctx context.Context) float64 {
	rate, err := s.bcv.GetRate(ctx)
	if err != nil {
		slog.Warn("bcv rate unavailable, delivery_fee_bs will be 0", "error", err)
		return 0
	}
	return rate.Rate
}

func applyBsRate(shops dtos.ShopsResponse, rate float64) {
	for i := range shops {
		shops[i].DeliveryFeeBs = money.RoundBs(shops[i].DeliveryFee * rate)
	}
}

func (s *shopsApp) GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error) {
	shops, err := s.repo.GetAllShops(ctx, coords)
	if err != nil {
		return nil, err
	}
	applyBsRate(shops, s.bcvRate(ctx))
	return shops, nil
}

func (p *shopsApp) GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error) {
	shops, err := p.repo.GetShopsBy(ctx, criteria, coords)
	if err != nil {
		return nil, err
	}
	applyBsRate(shops, p.bcvRate(ctx))
	return shops, nil
}

func (s *shopsApp) GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error) {
	return s.repo.GetShopByID(ctx, shopID)
}

func (s *shopsApp) SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error {
	if shopID == nil {
		return errcustom.ErrUnautorized
	}
	if err := s.repo.SetOpenClose(ctx, shopID, open); err != nil {
		return err
	}
	if err := s.broadcaster.BroadcastShopStatus(*shopID, open); err != nil {
		slog.Error("error al hacer broadcast del estado del local", "error", err, "shopID", shopID)
	}
	return nil
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

	rate := s.bcvRate(ctx)
	result := make(dtos.SearchShopsResult, 0, len(candidates))
	for _, c := range candidates {
		c.DeliveryFeeBs = money.RoundBs(c.DeliveryFee * rate)
		resp := dtos.SearchShopResponse{ShopResponse: c.ShopResponse}
		resp.MatchedProducts = matchedItems[c.ID]
		result = append(result, resp)
	}

	return result, nil
}
