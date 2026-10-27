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

// ShopEventPublisher publica eventos de shop al stream compartido (mismo
// contrato que orders.RedisStream), para que el ETL resincronice DimShop
// en tiempo real en vez de esperar al batch.
type ShopEventPublisher interface {
	PublisherEvent(ctx context.Context, data any) error
}

type ShopsApp interface {
	GetAllShops(ctx context.Context, query dtos.ListShopsQuery) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error)
	SearchShops(ctx context.Context, query dtos.SearchShopsQuery) (dtos.SearchShopsResult, error)
	SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error
	// GetLaunchPromoEndsAt expone la ventana de descuento día 1 a otros dominios
	// (products) sin que estos dependan de shops/domain/ports en general.
	GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error)
	// GetTurboEntry asigna el nodo Turbo abierto más cercano (≤2 km) a las
	// coordenadas dadas, o Available=false si no hay cobertura.
	GetTurboEntry(ctx context.Context, coords dtos.Coords) (dtos.TurboEntryResponse, error)
}

type ShopsRepo interface {
	GetAllShops(ctx context.Context, query dtos.ListShopsQuery) (dtos.ShopsResponse, error)
	GetShopsBy(ctx context.Context, criteria dtos.SearchShopsByID, coords dtos.Coords) (dtos.ShopsResponse, error)
	GetShopByID(ctx context.Context, shopID uuid.UUID) (dtos.ShopResponse, error)
	SearchBatchByShopID(ctx context.Context, shopID uuid.UUID) (uuid.UUID, error)
	SearchShopCandidates(ctx context.Context, query dtos.SearchShopsQuery) ([]dtos.SearchShopCandidate, error)
	FindMatchedProducts(ctx context.Context, shopIDs []uuid.UUID, q string) (map[uuid.UUID][]dtos.MatchedProductItem, error)
	SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error
	// GetShopStatus expone el estado de onboarding (draft/active) para el gate
	// de apertura en ShopsApp.SetOpenClose.
	GetShopStatus(ctx context.Context, shopID uuid.UUID) (string, error)
	GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error)
	// TryCloseOpenBatch cierra el lote abierto del shop si el local está cerrado
	// y no quedan órdenes activas. Idempotente (0 filas si no aplica).
	TryCloseOpenBatch(ctx context.Context, shopID uuid.UUID) error
	// TryCloseAllEligibleBatches cierra todos los lotes elegibles (safety net).
	TryCloseAllEligibleBatches(ctx context.Context) error
	// ListTurboCandidates retorna todos los shops type=Turbo con licencia
	// activa y coordenadas válidas (abiertos o no); la selección del nodo
	// (abierto, más cercano, ≤2 km) la resuelve la capa app.
	ListTurboCandidates(ctx context.Context) (dtos.ShopsResponse, error)
	// ResolveTurboShop calcula delivery_fee/time del nodo Turbo ya elegido
	// para las coordenadas del cliente.
	ResolveTurboShop(ctx context.Context, shopID uuid.UUID, coords dtos.Coords) (dtos.ShopResponse, error)
}
