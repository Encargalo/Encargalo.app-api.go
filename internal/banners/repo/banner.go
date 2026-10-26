package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/banners/domain/models"
	"Encargalo.app-api.go/internal/banners/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// activeBannersCacheKey es una sola clave global (no varía por shop/usuario):
// la app consulta esta lista en cada apertura del home, así que cachearla
// evita pegarle a Postgres por cada sesión activa.
const (
	activeBannersCacheKey = "banners:active"
	activeBannersCacheTTL = time.Hour
)

type bannersRepo struct {
	db    *bun.DB
	cache cache.Cache
}

func NewBannersRepo(db *bun.DB, cache cache.Cache) ports.BannersRepo {
	return &bannersRepo{db: db, cache: cache}
}

func (r *bannersRepo) invalidateActiveCache(ctx context.Context) {
	if err := r.cache.Clear(ctx, activeBannersCacheKey); err != nil {
		slog.Error("error al limpiar el cache de banners activos", "error", err)
	}
}

func (r *bannersRepo) CreateBanner(ctx context.Context, banner *models.Banner) error {
	banner.ID = id.Ensure(banner.ID)
	if _, err := r.db.NewInsert().Model(banner).Exec(ctx); err != nil {
		slog.Error("banners CreateBanner failed", "error", err)
		return errcustom.Unexpected(err)
	}
	r.invalidateActiveCache(ctx)
	return nil
}

func (r *bannersRepo) GetBannerByID(ctx context.Context, bannerID uuid.UUID) (*models.Banner, error) {
	var banner models.Banner
	err := r.db.NewSelect().Model(&banner).Where("id = ?", bannerID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("banners GetBannerByID failed", "bannerID", bannerID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return &banner, nil
}

func (r *bannersRepo) ListBanners(ctx context.Context) (models.Banners, error) {
	var banners models.Banners
	if err := r.db.NewSelect().Model(&banners).Order("position ASC", "created_at ASC").Scan(ctx); err != nil {
		slog.Error("banners ListBanners failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return banners, nil
}

func (r *bannersRepo) ListActiveBanners(ctx context.Context) (models.Banners, error) {
	cached, _ := r.cache.Get(ctx, activeBannersCacheKey)
	if cached != "" {
		var banners models.Banners
		if err := json.Unmarshal([]byte(cached), &banners); err == nil {
			return banners, nil
		}
	}

	var banners models.Banners
	if err := r.db.NewSelect().
		Model(&banners).
		Where("active = TRUE").
		Order("position ASC", "created_at ASC").
		Scan(ctx); err != nil {
		slog.Error("banners ListActiveBanners failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}

	if err := r.cache.Set(ctx, activeBannersCacheKey, banners, activeBannersCacheTTL); err != nil {
		slog.Error("error al guardar los banners activos en el cache", "error", err)
	}

	return banners, nil
}

func (r *bannersRepo) UpdateBanner(ctx context.Context, banner *models.Banner) error {
	_, err := r.db.NewUpdate().
		Model(banner).
		Column("image_url", "link_url", "position", "active").
		Set("updated_at = now()").
		WherePK().
		Exec(ctx)
	if err != nil {
		slog.Error("banners UpdateBanner failed", "bannerID", banner.ID, "error", err)
		return errcustom.Unexpected(err)
	}
	r.invalidateActiveCache(ctx)
	return nil
}

func (r *bannersRepo) DeleteBanner(ctx context.Context, bannerID uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*models.Banner)(nil)).Where("id = ?", bannerID).Exec(ctx)
	if err != nil {
		slog.Error("banners DeleteBanner failed", "bannerID", bannerID, "error", err)
		return errcustom.Unexpected(err)
	}
	r.invalidateActiveCache(ctx)
	return nil
}
