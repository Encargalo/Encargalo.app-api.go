package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"context"
	"database/sql"

	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsDTO "Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/entity"
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

	cacheTTL := 2 * time.Minute

	if cached, err := s.cache.Get(ctx, keyCache); err == nil && cached != "" {
		var resp shopsDTO.ShopsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return resp, nil
		}
		slog.Error("error al deserializar negocios del cache", "error", err)
	}

	var shops shops.Shops

	q := s.db.NewSelect().
		Model(&shops).
		Order("score DESC").
		Where("license_status = ?", "active")

	if config.Get().Environment.Current == "production" {
		q.Where("latitude IS NOT NULL AND longitude IS NOT NULL").
			Where(`
		6371 * acos(
			cos(radians(?)) * cos(radians(latitude)) *
			cos(radians(longitude) - radians(?)) +
			sin(radians(?)) * sin(radians(latitude))
		) <= coverage
	`, coords.Latitude, coords.Longitude, coords.Latitude)
	}

	if err := q.Scan(ctx); err != nil {
		slog.Error("error al obtener negocios", "error", err)
		return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
	}

	if len(shops) == 0 {
		return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
	}

	result, err := calculateDistances(coords, &shops)
	if err != nil {
		slog.Error("error al calcular distancias", "error", err)
	}

	resp := shops.ToDomainDTO(result)

	if err := s.cache.Set(ctx, keyCache, resp, cacheTTL); err != nil {
		slog.Error("error al guardar negocios en cache", "error", err)
	}

	return resp, nil
}

func (s *shopsRepo) GetShopsBy(ctx context.Context, criteria shopsDTO.SearchShopsByID, coords shopsDTO.Coords) (shopsDTO.ShopsResponse, error) {

	lat := strconv.FormatFloat(coords.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(coords.Longitude, 'f', -1, 64)
	keyCache := shops_all + criteria.Tag + lat + lon

	cacheTTL := 2 * time.Minute // 1*2*time.Minute → 2 min igual

	if cached, err := s.cache.Get(ctx, keyCache); err == nil && cached != "" {
		var resp shopsDTO.ShopsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return resp, nil
		}
		slog.Error("error al deserializar cache de negocios", "error", err)
	}

	var shops shops.Shops

	q := s.db.NewSelect().
		Model(&shops).
		Where("license_status = ?", "active")

	if criteria.ID != uuid.Nil {
		q.Where("id = ?", criteria.ID)
	}
	if criteria.Tag != "" {
		q.Where("tag = ?", criteria.Tag)
	}

	if config.Get().Environment.Current == "production" {
		q.Where("latitude IS NOT NULL AND longitude IS NOT NULL").
			Where(`
		6371 * acos(
			cos(radians(?)) * cos(radians(latitude)) *
			cos(radians(longitude) - radians(?)) +
			sin(radians(?)) * sin(radians(latitude))
		) <= coverage
	`, coords.Latitude, coords.Longitude, coords.Latitude)
	}

	if err := q.Scan(ctx); err != nil {
		slog.Error("error al obtener los negocios", "error", err)
		return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
	}

	if len(shops) == 0 {
		return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
	}

	result, err := calculateDistances(coords, &shops)
	if err != nil {
		slog.Error("error al calcular distancias", "error", err)
	}

	resp := shops.ToDomainDTO(result)

	if err := s.cache.Set(ctx, keyCache, resp, cacheTTL); err != nil {
		slog.Error("error al guardar negocios en cache", "error", err)
	}

	return resp, nil
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

func calculateDistances(coords shopsDTO.Coords, shopsResult *shops.Shops) ([]entity.ShopDistance, error) {

	origins := fmt.Sprintf("%f,%f", coords.Latitude, coords.Longitude)

	var shopIDs []uuid.UUID
	var destinationsRaw []string

	for _, shop := range *shopsResult {
		if shop.Latitude != nil && shop.Longitude != nil {
			shopIDs = append(shopIDs, shop.ID)
			destinationsRaw = append(destinationsRaw, fmt.Sprintf("%f,%f", *shop.Latitude, *shop.Longitude))
		}
	}

	if len(destinationsRaw) == 0 {
		return nil, errors.New("no shops with valid coordinates")
	}

	if len(destinationsRaw) > 25 {
		return nil, fmt.Errorf("too many destinations: %d (max 25)", len(destinationsRaw))
	}

	destinationsJoined := strings.Join(destinationsRaw, "|")

	params := url.Values{}
	params.Set("origins", origins)
	params.Set("destinations", destinationsJoined)
	params.Set("mode", "driving")
	params.Set("key", config.Get().Google.Matrix)

	fullURL := "https://maps.googleapis.com/maps/api/distancematrix/json?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error calling google api: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if len(bodyBytes) == 0 {
		return nil, errors.New("empty response from google")
	}

	// Decodificar JSON
	var data struct {
		Status string `json:"status"`
		Rows   []struct {
			Elements []struct {
				Distance struct {
					Value float64 `json:"value"`
				} `json:"distance"`
				Duration struct {
					Value int64 `json:"value"`
				} `json:"duration"`
				Status string `json:"status"`
			} `json:"elements"`
		} `json:"rows"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("decode json error: %w; body=%s", err, string(bodyBytes))
	}

	if data.Status != "OK" {
		return nil, fmt.Errorf("google api status=%s", data.Status)
	}

	if len(data.Rows) == 0 || len(data.Rows[0].Elements) == 0 {
		return nil, errors.New("invalid google matrix response")
	}

	elements := data.Rows[0].Elements

	// Crear result, usando los IDs paralelos
	result := make([]entity.ShopDistance, 0, len(shopIDs))

	for i := range shopIDs {
		if i >= len(elements) {
			break
		}
		elem := elements[i]

		if elem.Status != "OK" {
			continue
		}

		result = append(result, entity.ShopDistance{
			ShopID:   shopIDs[i],
			Distance: elem.Distance.Value,
			Duration: elem.Duration.Value / 60,
		})
	}

	return result, nil
}
