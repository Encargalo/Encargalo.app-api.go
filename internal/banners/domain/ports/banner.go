package ports

import (
	"context"

	"Encargalo.app-api.go/internal/banners/domain/dtos"
	"Encargalo.app-api.go/internal/banners/domain/models"
	"github.com/google/uuid"
)

// BannersRepo persiste los banners publicitarios. ListActiveBanners es la
// única ruta que pasa por cache (cache-aside en Redis, TTL 1h): es la que
// consume la app en cada apertura del home, mientras que ListBanners (staff)
// necesita ver el estado real incluyendo inactivos recién tocados.
type BannersRepo interface {
	CreateBanner(ctx context.Context, banner *models.Banner) error
	GetBannerByID(ctx context.Context, id uuid.UUID) (*models.Banner, error)
	ListBanners(ctx context.Context) (models.Banners, error)
	ListActiveBanners(ctx context.Context) (models.Banners, error)
	UpdateBanner(ctx context.Context, banner *models.Banner) error
	DeleteBanner(ctx context.Context, id uuid.UUID) error
}

type BannersApp interface {
	CreateBanner(ctx context.Context, req dtos.CreateBannerRequest) (dtos.BannerResponse, error)
	ListBanners(ctx context.Context) (dtos.BannersResponse, error)
	GetBanner(ctx context.Context, id uuid.UUID) (dtos.BannerResponse, error)
	UpdateBanner(ctx context.Context, id uuid.UUID, req dtos.UpdateBannerRequest) (dtos.BannerResponse, error)
	DeleteBanner(ctx context.Context, id uuid.UUID) error
	ListActiveBanners(ctx context.Context) (dtos.BannersResponse, error)
}
