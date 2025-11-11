package repo

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"context"
	"database/sql"

	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsDTO "Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/models/shops"
	ports "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	shops_all = "shops_all_coords"
)

type shopsRepo struct {
	db    *bun.DB
	cache cache.Cache
}

func NewShopsRepository(db *bun.DB, cache cache.Cache) ports.ShopsRepo {
	return &shopsRepo{db, cache}
}

func (s *shopsRepo) GetAllShops(ctx context.Context, coords shopsDTO.Coords) (shopsDTO.ShopsResponse, error) {

	lat := strconv.FormatFloat(coords.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(coords.Longitude, 'f', -1, 64)
	keyCache := shops_all + lat + lon
	timeCache := 1 * 2 * time.Minute

	var shops shops.Shops

	if *config.Get().Limit.Status {

		shopCache, err := s.cache.Get(ctx, keyCache)
		if err != nil {
			slog.Error("error al obtener los negocios del cache", "error", err)
		}

		if shopCache != "" {
			var shops shopsDTO.ShopsResponse
			if err := json.Unmarshal([]byte(shopCache), &shops); err != nil {
				slog.Error("error al deserializar los negocios del cache", "error", err)
			}
			return shops, nil
		}

		if err := s.db.NewSelect().Model(&shops).Order("score DESC").
			Where("license_status = ?", "active").
			Where(`
			6371 * acos(
				cos(radians(?)) * cos(radians(latitude)) *
				cos(radians(longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(latitude))
			) <= 4.5
		`, coords.Latitude, coords.Longitude, coords.Latitude).
			Scan(ctx); err != nil {
			slog.Error("error al obtener los negocios", "error", err)
			return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
		}

		if len(shops) == 0 {
			return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
		}

		if err = s.cache.Set(ctx, keyCache, shops.ToDomainDTO(), timeCache); err != nil {
			slog.Error("error al guardar los negocios en el cache", "error", err)
		}

		return shops.ToDomainDTO(), nil
	}

	shopCache, err := s.cache.Get(ctx, keyCache)
	if err != nil {
		slog.Error("error al obtener los negocios del cache", "error", err)
	}

	if shopCache != "" {
		var shops shopsDTO.ShopsResponse
		if err := json.Unmarshal([]byte(shopCache), &shops); err != nil {
			slog.Error("error al deserializar los negocios del cache", "error", err)
		}
		return shops, nil
	}

	if err := s.db.NewSelect().Model(&shops).Order("score DESC").
		Where("license_status = ?", "active").Scan(ctx); err != nil {
		slog.Error("error al obtener los negocios", "error", err)
		return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
	}

	if len(shops) == 0 {
		return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
	}

	if err = s.cache.Set(ctx, keyCache, shops.ToDomainDTO(), timeCache); err != nil {
		slog.Error("error al guardar los negocios en el cache", "error", err)
	}

	return shops.ToDomainDTO(), nil
}

func (s *shopsRepo) GetShopsBy(ctx context.Context, criteria shopsDTO.SearchShopsByID, coords shopsDTO.Coords) (shopsDTO.ShopsResponse, error) {

	lat := strconv.FormatFloat(coords.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(coords.Longitude, 'f', -1, 64)
	keyCache := shops_all + criteria.Tag + lat + lon
	timeCache := 1 * 2 * time.Minute

	var shops shops.Shops

	if *config.Get().Limit.Status {

		shopCache, err := s.cache.Get(ctx, keyCache)
		if err != nil {
			slog.Error("error al obtener los negocios del cache", "error", err)
		}

		if shopCache != "" {
			var shops shopsDTO.ShopsResponse
			if err := json.Unmarshal([]byte(shopCache), &shops); err != nil {
				slog.Error("error al deserializar los negocios del cache", "error", err)
			}
			return shops, nil
		}

		if err := s.db.NewSelect().
			Model(&shops).
			Where("license_status = ?", "active").
			Where(`
			6371 * acos(
				cos(radians(?)) * cos(radians(latitude)) *
				cos(radians(longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(latitude))
			) <= 4.5
		`, coords.Latitude, coords.Longitude, coords.Latitude).
			WhereGroup("and", func(sq *bun.SelectQuery) *bun.SelectQuery {
				if criteria.ID != uuid.Nil {
					sq = sq.Where("id = ?", criteria.ID)
				}
				if criteria.Tag != "" {
					sq = sq.Where("tag = ?", criteria.Tag)
				}
				return sq
			}).
			Scan(ctx); err != nil {
			slog.Error("error al obtener los negocios", "error", err)
			return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
		}

		if len(shops) == 0 {
			return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
		}

		if err = s.cache.Set(ctx, keyCache, shops.ToDomainDTO(), timeCache); err != nil {
			slog.Error("error al guardar los negocios en el cache", "error", err)
		}

		return shops.ToDomainDTO(), nil
	}

	shopCache, err := s.cache.Get(ctx, keyCache)
	if err != nil {
		slog.Error("error al obtener los negocios del cache", "error", err)
	}

	if shopCache != "" {
		var shops shopsDTO.ShopsResponse
		if err := json.Unmarshal([]byte(shopCache), &shops); err != nil {
			slog.Error("error al deserializar los negocios del cache", "error", err)
		}
		return shops, nil
	}

	if err := s.db.NewSelect().
		Model(&shops).
		Where("license_status = ?", "active").
		WhereGroup("and", func(sq *bun.SelectQuery) *bun.SelectQuery {
			if criteria.ID != uuid.Nil {
				sq = sq.Where("id = ?", criteria.ID)
			}
			if criteria.Tag != "" {
				sq = sq.Where("tag = ?", criteria.Tag)
			}
			return sq
		}).
		Scan(ctx); err != nil {
		slog.Error("error al obtener los negocios", "error", err)
		return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
	}

	if len(shops) == 0 {
		return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
	}

	if err = s.cache.Set(ctx, keyCache, shops.ToDomainDTO(), timeCache); err != nil {
		slog.Error("error al guardar los negocios en el cache", "error", err)
	}

	return shops.ToDomainDTO(), nil
}

func (s *shopsRepo) SearchBatchByShopID(ctx context.Context, shopID uuid.UUID) (uuid.UUID, error) {

	var batchID uuid.UUID

	if err := s.db.NewSelect().Table("business.batches").Column("id").Where("shop_id = ? and opened = ?", shopID, true).
		Scan(ctx, &batchID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("error al obtener el batch por shop id", "error", err)
			return uuid.Nil, errcustom.ErrUnexpectedError
		}
		return uuid.Nil, errcustom.ErrNotFound
	}

	return batchID, nil
}
