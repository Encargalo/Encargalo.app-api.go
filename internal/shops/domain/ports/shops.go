package ports

import (
	"context"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
)

type ShopsApp interface {
	GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error)
	SearchShops(ctx context.Context, query dtos.SearchShopsQuery) (dtos.SearchShopsResult, error)
}

type ShopsRepo interface {
	GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error)
	SearchBatchByShopID(ctx context.Context, shopID uuid.UUID) (uuid.UUID, error)
	SearchShopCandidates(ctx context.Context, query dtos.SearchShopsQuery) ([]dtos.SearchShopCandidate, error)
	FindMatchedProducts(ctx context.Context, shopIDs []uuid.UUID, q string) (map[uuid.UUID][]dtos.MatchedProductItem, error)
}
