package app

import (
	"context"
	"log/slog"
	"sort"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	portsShops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

// closedShopsOpeningWithinMinutes es la ventana usada por GetAllShops para
// decidir qué locales cerrados se siguen mostrando (solo los que abren
// pronto), al final del listado.
const closedShopsOpeningWithinMinutes = 30 * time.Minute

type shopsApp struct {
	repo        portsShops.ShopsRepo
	broadcaster portsShops.ShopStatusBroadcaster
	bcv         bcvPorts.BcvApp
	events      portsShops.ShopEventPublisher
}

func NewShopsApp(repo portsShops.ShopsRepo, broadcaster portsShops.ShopStatusBroadcaster, bcv bcvPorts.BcvApp, events portsShops.ShopEventPublisher) portsShops.ShopsApp {
	return &shopsApp{repo, broadcaster, bcv, events}
}

func (s *shopsApp) publishShopUpdated(ctx context.Context, shopID uuid.UUID) {
	if s.events == nil {
		return
	}
	event := shopsmodels.DataEventShopUpdated{}
	event.BuildFromShopID(shopID.String())
	if err := s.events.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing shop_updated event", "shopID", shopID, "error", err)
	}
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

func (s *shopsApp) GetAllShops(ctx context.Context, query dtos.ListShopsQuery) (dtos.ShopsResponse, error) {
	shops, err := s.repo.GetAllShops(ctx, query)
	if err != nil {
		return nil, err
	}
	applyBsRate(shops, s.bcvRate(ctx))
	return s.enrichWithSchedule(ctx, shops, true)
}

func (p *shopsApp) GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error) {
	shops, err := p.repo.GetShopsBy(ctx, criteria, coords)
	if err != nil {
		return nil, err
	}
	applyBsRate(shops, p.bcvRate(ctx))
	return p.enrichWithSchedule(ctx, shops, false)
}

// enrichWithSchedule añade opens_at/closes_at a cada shop a partir de su
// horario semanal (business.schedules) y, si filterAndSort, descarta los
// locales cerrados que no abren pronto (≤30min) y ordena abiertos primero,
// cerrados-que-abren-pronto al final (por opens_at ascendente).
func (s *shopsApp) enrichWithSchedule(ctx context.Context, shops dtos.ShopsResponse, filterAndSort bool) (dtos.ShopsResponse, error) {
	if len(shops) == 0 {
		return shops, nil
	}

	shopIDs := make([]uuid.UUID, len(shops))
	for i, shop := range shops {
		shopIDs[i] = shop.ID
	}

	schedulesByShop, err := s.repo.GetSchedulesByShopIDs(ctx, shopIDs)
	if err != nil {
		slog.Error("error al obtener horario de locales, se omite opens_at/closes_at", "error", err)
		schedulesByShop = map[uuid.UUID][]dtos.ScheduleResponse{}
	}

	now := bizclock.Now()

	if !filterAndSort {
		for i := range shops {
			opensAt, closesAt := bizclock.NextTransition(toBizclockWindows(schedulesByShop[shops[i].ID]), now)
			if shops[i].Opened {
				shops[i].ClosesAt = closesAt
			} else {
				shops[i].OpensAt = opensAt
			}
		}
		return shops, nil
	}

	open := make(dtos.ShopsResponse, 0, len(shops))
	closedSoon := make(dtos.ShopsResponse, 0)

	for _, shop := range shops {
		opensAt, closesAt := bizclock.NextTransition(toBizclockWindows(schedulesByShop[shop.ID]), now)

		if shop.Opened {
			shop.ClosesAt = closesAt
			open = append(open, shop)
			continue
		}

		if opensAt != nil && !opensAt.After(now.Add(closedShopsOpeningWithinMinutes)) {
			shop.OpensAt = opensAt
			closedSoon = append(closedSoon, shop)
		}
	}

	sort.SliceStable(closedSoon, func(i, j int) bool {
		return closedSoon[i].OpensAt.Before(*closedSoon[j].OpensAt)
	})

	return append(open, closedSoon...), nil
}

func toBizclockWindows(schedules []dtos.ScheduleResponse) []bizclock.Window {
	windows := make([]bizclock.Window, len(schedules))
	for i, s := range schedules {
		dayOfWeek := s.DayOfWeek
		windows[i] = bizclock.Window{
			DayOfWeek: &dayOfWeek,
			StartTime: s.Open,
			EndTime:   s.Close,
			Enabled:   s.Enabled,
		}
	}
	return windows
}

func (s *shopsApp) GetMySchedule(ctx context.Context, shopID uuid.UUID) ([]dtos.ScheduleResponse, error) {
	return s.repo.GetShopSchedule(ctx, shopID)
}

func (s *shopsApp) SetMySchedule(ctx context.Context, shopID uuid.UUID, schedules []dtos.ScheduleInput) error {
	return s.repo.ReplaceShopSchedule(ctx, shopID, schedules)
}

func (s *shopsApp) GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error) {
	return s.repo.GetShopByID(ctx, shopID)
}

func (s *shopsApp) SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error {
	if shopID == nil {
		return errcustom.ErrUnautorized
	}
	if open {
		status, err := s.repo.GetShopStatus(ctx, *shopID)
		if err != nil {
			return err
		}
		if status != string(shopsmodels.ShopStatusActive) {
			return errcustom.ErrShopNotActive
		}
	}
	if err := s.repo.SetOpenClose(ctx, shopID, open); err != nil {
		return err
	}
	if !open {
		// Si ya no hay órdenes activas, cierra el lote de inmediato.
		if err := s.repo.TryCloseOpenBatch(ctx, *shopID); err != nil {
			slog.Error("error al intentar cerrar lote tras cerrar local", "error", err, "shopID", shopID)
		}
	}
	if err := s.broadcaster.BroadcastShopStatus(*shopID, open); err != nil {
		slog.Error("error al hacer broadcast del estado del local", "error", err, "shopID", shopID)
	}
	s.publishShopUpdated(ctx, *shopID)
	return nil
}

func (s *shopsApp) GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error) {
	return s.repo.GetLaunchPromoEndsAt(ctx, shopID)
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
	matchedItems = filterMatchedItemsBySchedule(matchedItems, bizclock.Now())

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

func filterMatchedItemsBySchedule(matched map[uuid.UUID][]dtos.MatchedProductItem, now time.Time) map[uuid.UUID][]dtos.MatchedProductItem {
	filtered := make(map[uuid.UUID][]dtos.MatchedProductItem, len(matched))
	for shopID, items := range matched {
		kept := make([]dtos.MatchedProductItem, 0, len(items))
		for _, item := range items {
			if !bizclock.IsAvailableAt(toShopsWindows(item.Schedules), now) {
				continue
			}
			kept = append(kept, item)
		}
		if len(kept) > 0 {
			filtered[shopID] = kept
		}
	}
	return filtered
}

func toShopsWindows(schedules []dtos.MatchedSchedule) []bizclock.Window {
	windows := make([]bizclock.Window, len(schedules))
	for i, s := range schedules {
		windows[i] = bizclock.Window{
			DayOfWeek: s.DayOfWeek,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			Enabled:   s.Enabled,
		}
	}
	return windows
}
