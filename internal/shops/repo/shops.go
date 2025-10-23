package repo

import (
	"errors"

	"context"
	"database/sql"
	"fmt"

	"Encargalo.app-api.go/internal/shared/config"
	shopsDTO "Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/models/shops"
	ports "Encargalo.app-api.go/internal/shops/domain/ports/shops"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type shopsRepo struct {
	db *bun.DB
}

func NewShopsRepository(db *bun.DB) ports.ShopsRepo {
	return &shopsRepo{db}
}

func (s *shopsRepo) GetAllShops(ctx context.Context, coords shopsDTO.Coords) (shopsDTO.ShopsResponse, error) {

	var shops shops.Shops

	if config.Get().Limit.Status {
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
			if errors.Is(err, sql.ErrNoRows) {
				return shopsDTO.ShopsResponse{}, errors.New("not found")
			}
			fmt.Println(err.Error())
			return shopsDTO.ShopsResponse{}, errors.New("unexpected error")
		}

		return shops.ToDomainDTO(), nil
	}

	if err := s.db.NewSelect().Model(&shops).Order("score DESC").
		Where("license_status = ?", "active").Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shopsDTO.ShopsResponse{}, errors.New("not found")
		}
		fmt.Println(err.Error())
		return shopsDTO.ShopsResponse{}, errors.New("unexpected error")
	}

	return shops.ToDomainDTO(), nil
}

func (p *shopsRepo) GetShopsBy(ctx context.Context, criteria shopsDTO.SearchShopsByID, coords shopsDTO.Coords) (shopsDTO.ShopsResponse, error) {

	products := new(shops.Shops)

	if config.Get().Limit.Status {
		if err := p.db.NewSelect().
			Model(products).
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
			if errors.Is(err, sql.ErrNoRows) {
				return shopsDTO.ShopsResponse{}, errors.New("not found")
			}
			fmt.Println(err.Error())
			return shopsDTO.ShopsResponse{}, errors.New("unexpected error")
		}
		return products.ToDomainDTO(), nil
	}

	if err := p.db.NewSelect().
		Model(products).
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
		if errors.Is(err, sql.ErrNoRows) {
			return shopsDTO.ShopsResponse{}, errors.New("not found")
		}
		fmt.Println(err.Error())
		return shopsDTO.ShopsResponse{}, errors.New("unexpected error")
	}
	return products.ToDomainDTO(), nil
}
