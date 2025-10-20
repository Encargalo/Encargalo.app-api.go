package ports

import (
	"context"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
)

type ShopsApp interface {
	GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
}

type ShopsRepo interface {
	GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
}
