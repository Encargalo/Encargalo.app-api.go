package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
)

// ShopStatusBroadcaster notifica a los clientes conectados el nuevo estado del local.
// Lo implementa orders/app.appManagerOrder sin importar este paquete.
type ShopStatusBroadcaster interface {
	BroadcastShopStatus(shopID uuid.UUID, open bool) error
}

type ShopsApp interface {
	GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error)
	SearchShops(ctx context.Context, query dtos.SearchShopsQuery) (dtos.SearchShopsResult, error)
	SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error
	// GetLaunchPromoEndsAt expone la ventana de descuento día 1 a otros dominios
	// (products) sin que estos dependan de shops/domain/ports en general.
	GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error)
}

type ShopsRepo interface {
	GetAllShops(ctx context.Context, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error)
	SearchBatchByShopID(ctx context.Context, shopID uuid.UUID) (uuid.UUID, error)
	SearchShopCandidates(ctx context.Context, query dtos.SearchShopsQuery) ([]dtos.SearchShopCandidate, error)
	FindMatchedProducts(ctx context.Context, shopIDs []uuid.UUID, q string) (map[uuid.UUID][]dtos.MatchedProductItem, error)
	SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error
	GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error)
}
