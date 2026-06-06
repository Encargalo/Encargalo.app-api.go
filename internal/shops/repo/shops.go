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
	productModels "Encargalo.app-api.go/internal/products/domain/models"
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
	shops_all = "shops:"
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

	var resp shopsDTO.ShopsResponse

	if config.Get().Environment.Current == "production" {
		result, err := calculateDistances(coords, &shops)
		if err != nil {
			slog.Error("error al calcular distancias", "error", err)
		}

		resp = shops.ToDomainDTO(result)
	} else {

		dataDistance := make([]entity.ShopDistance, len(shops))

		for i, v := range shops {
			dataDistance[i].ShopID = v.ID
			dataDistance[i].Distance = 0
			dataDistance[i].Duration = 0
		}

		resp = shops.ToDomainDTO(dataDistance)
	}

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

	var resp shopsDTO.ShopsResponse

	if config.Get().Environment.Current == "production" {
		result, err := calculateDistances(coords, &shops)
		if err != nil {
			slog.Error("error al calcular distancias", "error", err)
		}

		resp = shops.ToDomainDTO(result)
	} else {

		dataDistance := make([]entity.ShopDistance, len(shops))

		for i, v := range shops {
			dataDistance[i].ShopID = v.ID
			dataDistance[i].Distance = 0
			dataDistance[i].Duration = 0
		}

		resp = shops.ToDomainDTO(dataDistance)
	}

	if err := s.cache.Set(ctx, keyCache, resp, cacheTTL); err != nil {
		slog.Error("error al guardar negocios en cache", "error", err)
	}

	return resp, nil
}

func (s *shopsRepo) GetShopByID(ctx context.Context, shopID uuid.UUID) (shopsDTO.ShopResponse, error) {
	var shop shops.Shop
	if err := s.db.NewSelect().Model(&shop).Where("id = ?", shopID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shopsDTO.ShopResponse{}, errcustom.ErrNotFound
		}
		slog.Error("error al obtener el negocio por id", "error", err)
		return shopsDTO.ShopResponse{}, errcustom.ErrUnexpectedError
	}

	var lat, lon float64
	if shop.Latitude != nil {
		lat = *shop.Latitude
	}
	if shop.Longitude != nil {
		lon = *shop.Longitude
	}

	return shopsDTO.ShopResponse{
		ID:        shop.ID,
		Name:      shop.Name,
		Tag:       shop.Tag,
		Address:   shop.Address,
		Latitude:  lat,
		Longitude: lon,
		HomePhone: shop.HomePhone,
		Logo:      shop.LogoImage,
		Banner:    shop.Banner,
		Opened:    shop.Opened,
		Type:      shop.Type,
		Score:     shop.Score,
	}, nil
}

func (s *shopsRepo) SearchShopCandidates(ctx context.Context, query shopsDTO.SearchShopsQuery) ([]shopsDTO.SearchShopCandidate, error) {
	qNorm := strings.ToLower(strings.TrimSpace(query.Q))
	lat4 := strconv.FormatFloat(query.Latitude, 'f', 4, 64)
	lon4 := strconv.FormatFloat(query.Longitude, 'f', 4, 64)
	keyCache := "shops:search:" + qNorm + ":" + lat4 + ":" + lon4
	cacheTTL := 2 * time.Minute

	if cached, err := s.cache.Get(ctx, keyCache); err == nil && cached != "" {
		var resp []shopsDTO.SearchShopCandidate
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return resp, nil
		}
		slog.Error("error al deserializar búsqueda del cache", "error", err)
	}

	rawSQL := `
		SELECT
			s.id, s.name, s.tag, s.address, s.latitude, s.longitude,
			s.home_phone, s.logo_image, s.banner, s.opened, s.type,
			s.score, s.coverage, s.minimum_delivery, s.maximum_delivery,
			(
				to_tsvector('spanish', s.name) @@ plainto_tsquery('spanish', ?)
				OR s.name ILIKE '%' || ? || '%'
			) AS matched_by_name
		FROM business.shops s
		WHERE s.opened = true
		  AND s.license_status = 'active'
		  AND s.latitude IS NOT NULL
		  AND s.longitude IS NOT NULL
		  AND s.deleted_at IS NULL
		  AND 6371 * acos(
				cos(radians(?)) * cos(radians(s.latitude)) *
				cos(radians(s.longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(s.latitude))
			) <= 3
		  AND (
			  to_tsvector('spanish', s.name) @@ plainto_tsquery('spanish', ?)
			  OR s.name ILIKE '%' || ? || '%'
			  OR EXISTS (
				  SELECT 1 FROM products.categories cat
				  LEFT JOIN products.macro_categories mc ON mc.id = cat.macro_category_id
				  WHERE cat.shop_id = s.id
					AND (
						to_tsvector('spanish', cat.name) @@ plainto_tsquery('spanish', ?)
						OR cat.name ILIKE '%' || ? || '%'
						OR to_tsvector('spanish', mc.name) @@ plainto_tsquery('spanish', ?)
						OR mc.name ILIKE '%' || ? || '%'
					)
			  )
		  )
		ORDER BY s.score DESC
	`

	var rows []shops.SearchShopRow
	if err := s.db.NewRaw(
		rawSQL,
		qNorm, qNorm,
		query.Latitude, query.Longitude, query.Latitude,
		qNorm, qNorm, qNorm, qNorm, qNorm, qNorm,
	).Scan(ctx, &rows); err != nil {
		slog.Error("error en búsqueda de tiendas", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if len(rows) == 0 {
		return nil, nil
	}

	shopModels := make(shops.Shops, len(rows))
	for i, row := range rows {
		shopModels[i] = shops.Shop{
			ID:              row.ID,
			Name:            row.Name,
			Tag:             row.Tag,
			Address:         row.Address,
			HomePhone:       row.HomePhone,
			LogoImage:       row.LogoImage,
			Banner:          row.Banner,
			Opened:          row.Opened,
			Type:            row.Type,
			Score:           row.Score,
			Latitude:        row.Latitude,
			Longitude:       row.Longitude,
			Coverage:        row.Coverage,
			MinimunDelivery: row.MinimumDelivery,
			MaximumDelivery: row.MaximumDelivery,
		}
	}

	distances := make([]entity.ShopDistance, len(rows))
	for i, row := range rows {
		distances[i] = entity.ShopDistance{ShopID: row.ID}
	}

	if config.Get().Environment.Current == "production" {
		coords := shopsDTO.Coords{Latitude: query.Latitude, Longitude: query.Longitude}
		prodDistances, err := calculateDistances(coords, &shopModels)
		if err != nil {
			slog.Error("error al calcular distancias en búsqueda", "error", err)
			return nil, errcustom.ErrUnexpectedError
		}
		distances = prodDistances
	}

	distMap := make(map[uuid.UUID]entity.ShopDistance, len(distances))
	for _, d := range distances {
		distMap[d.ShopID] = d
	}

	candidates := make([]shopsDTO.SearchShopCandidate, 0, len(rows))
	for i, row := range rows {
		dist, ok := distMap[row.ID]
		if !ok {
			continue
		}
		candidates = append(candidates, shopsDTO.SearchShopCandidate{
			ShopResponse:  shopModels[i].ToDomainDTO(dist),
			MatchedByName: row.MatchedByName,
		})
	}

	if err := s.cache.Set(ctx, keyCache, candidates, cacheTTL); err != nil {
		slog.Error("error al guardar búsqueda en cache", "error", err)
	}

	return candidates, nil
}

func (s *shopsRepo) FindMatchedProducts(ctx context.Context, shopIDs []uuid.UUID, q string) (map[uuid.UUID][]shopsDTO.MatchedProductItem, error) {
	qNorm := strings.ToLower(strings.TrimSpace(q))

	var items []productModels.Item
	if err := s.db.NewSelect().
		Model(&items).
		Relation("ItemRule").
		Join("JOIN products.categories AS cat ON cat.id = item.category_id").
		Join("LEFT JOIN products.macro_categories AS mc ON mc.id = cat.macro_category_id").
		Where("item.shop_id IN (?)", bun.In(shopIDs)).
		Where("item.is_available = true").
		Where("item.deleted_at IS NULL").
		Where("(to_tsvector('spanish', cat.name) @@ plainto_tsquery('spanish', ?) OR cat.name ILIKE ? OR to_tsvector('spanish', mc.name) @@ plainto_tsquery('spanish', ?) OR mc.name ILIKE ?)",
			qNorm, "%"+qNorm+"%", qNorm, "%"+qNorm+"%").
		Scan(ctx); err != nil {
		slog.Error("error al cargar productos del buscador", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	result := make(map[uuid.UUID][]shopsDTO.MatchedProductItem, len(shopIDs))
	for _, item := range items {
		rules := make([]shopsDTO.MatchedItemRule, 0, len(item.ItemRule))
		for _, r := range item.ItemRule {
			rules = append(rules, shopsDTO.MatchedItemRule{
				ID:           r.ID,
				ItemID:       r.ItemID,
				RuleKey:      r.RuleKey,
				RuleValue:    r.RuleValue,
				SelectorType: r.SelectorType,
			})
		}
		result[item.ShopID] = append(result[item.ShopID], shopsDTO.MatchedProductItem{
			ID:          item.ID,
			ShopID:      item.ShopID,
			Name:        item.Name,
			Ref:         float64(item.Price),
			Image:       item.Image,
			Description: item.Description,
			Score:       item.Score,
			HasFlavors:  item.HasFlavors,
			Rules:       rules,
		})
	}

	return result, nil
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
	defer func() { _ = resp.Body.Close() }()

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
