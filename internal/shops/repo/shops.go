package repo

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"context"
	"database/sql"

	"Encargalo.app-api.go/internal/pkg/cache"
	productModels "Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsDTO "Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/entity"
	shopsModels "Encargalo.app-api.go/internal/shops/domain/models"
	"Encargalo.app-api.go/internal/shops/domain/models/shops"
	ports "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	shops_all = "shops:"

	// launchPromoWindow y zeroCommissionWindow son horas corridas desde la
	// primera apertura del negocio (sin días calendario ni zona horaria):
	// día 1 con 50% de descuento al cliente, y comisión 0% durante 8 días
	// (192h = 8 x 24h). La ventana de día 1 siempre queda anidada dentro de
	// la de 8 días porque parten del mismo instante.
	launchPromoWindow    = 24 * time.Hour
	zeroCommissionWindow = 192 * time.Hour
)

type shopsRepo struct {
	db    *bun.DB
	cache cache.Cache
}

func NewShopsRepository(db *bun.DB, cache cache.Cache) ports.ShopsRepo {
	return &shopsRepo{db, cache}
}

// coverageRadiusKm resuelve el radio de listado/búsqueda según el type
// solicitado: Turbo usa un radio menor que Restaurant/Supermarket.
func coverageRadiusKm(shopType string) float64 {
	if shopType == shops.ShopTypeTurbo {
		return shops.TurboCoverageRadiusKm
	}
	return shops.CoverageRadiusKm
}

// devTimeMinutes es el tiempo fijo mostrado fuera de production (sin llamar
// a Google Maps). Turbo usa un valor menor porque no suma prep fija.
func devTimeMinutes(shopType string) int {
	if shopType == shops.ShopTypeTurbo {
		return 8
	}
	return 12
}

func (s *shopsRepo) GetAllShops(ctx context.Context, query shopsDTO.ListShopsQuery) (shopsDTO.ShopsResponse, error) {

	lat := strconv.FormatFloat(query.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(query.Longitude, 'f', -1, 64)
	keyCache := shops_all + lat + lon
	if query.Type != "" {
		keyCache += ":type:" + query.Type
	}

	cacheTTL := 2 * time.Minute

	if cached, err := s.cache.Get(ctx, keyCache); err == nil && cached != "" {
		var resp shopsDTO.ShopsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return resp, nil
		}
		slog.Error("error al deserializar negocios del cache", "error", err)
	}

	turboType := shops.ShopTypeTurbo
	radiusKm := coverageRadiusKm(query.Type)
	activeStatus := shops.ShopStatusActive

	var shops shops.Shops

	q := s.db.NewSelect().
		Model(&shops).
		Order("score DESC").
		Where("license_status = ?", "active").
		Where("status = ?", activeStatus)

	if query.Type != "" {
		q = q.Where("type = ?", query.Type)
	}
	if query.Type == "" {
		// Sin type, el feed general excluye Turbo: se descubre vía /turbo/entry.
		q = q.Where("type != ?", turboType)
	}

	coords := query.Coords()

	if config.Get().IsProduction() {
		q.Where("latitude IS NOT NULL AND longitude IS NOT NULL").
			Where(`
		6371 * acos(
			cos(radians(?)) * cos(radians(latitude)) *
			cos(radians(longitude) - radians(?)) +
			sin(radians(?)) * sin(radians(latitude))
		) <= ?
	`, coords.Latitude, coords.Longitude, coords.Latitude, radiusKm)
	}

	if err := q.Scan(ctx); err != nil {
		slog.Error("error al obtener negocios", "error", err)
		return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
	}

	if len(shops) == 0 {
		return shopsDTO.ShopsResponse{}, errcustom.ErrNotFound
	}

	var resp shopsDTO.ShopsResponse

	if config.Get().IsProduction() {
		result, err := calculateDistances(coords, &shops)
		if err != nil {
			slog.Error("error al calcular distancias", "error", err)
		}

		resp = shops.ToDomainDTO(result)
	} else {
		dataDistance := make([]entity.ShopDistance, len(shops))
		for i, v := range shops {
			dataDistance[i].ShopID = v.ID
		}
		resp = shops.ToDomainDTO(dataDistance)
		for i := range resp {
			resp[i].DeliveryFee = deliveryfee.DevPlaceholderFee
			resp[i].Time = devTimeMinutes(resp[i].Type)
		}
	}

	if err := s.cache.Set(ctx, keyCache, resp, cacheTTL); err != nil {
		slog.Error("error al guardar negocios en cache", "error", err)
	}

	return resp, nil
}

func (s *shopsRepo) GetShopsBy(ctx context.Context, criteria shopsDTO.SearchShopsByID, coords shopsDTO.Coords) (shopsDTO.ShopsResponse, error) {

	lat := strconv.FormatFloat(coords.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(coords.Longitude, 'f', -1, 64)
	keyCache := shops_all + criteria.Slug + lat + lon

	cacheTTL := 2 * time.Minute // 1*2*time.Minute → 2 min igual

	if cached, err := s.cache.Get(ctx, keyCache); err == nil && cached != "" {
		var resp shopsDTO.ShopsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return resp, nil
		}
		slog.Error("error al deserializar cache de negocios", "error", err)
	}

	activeStatus := shops.ShopStatusActive

	var shops shops.Shops

	q := s.db.NewSelect().
		Model(&shops).
		Where("license_status = ?", "active").
		Where("status = ?", activeStatus)

	if criteria.ID != uuid.Nil {
		q.Where("id = ?", criteria.ID)
	}
	if criteria.Slug != "" {
		q.Where("slug = ?", criteria.Slug)
	}

	if config.Get().IsProduction() {
		q.Where("latitude IS NOT NULL AND longitude IS NOT NULL").
			Where(`
		6371 * acos(
			cos(radians(?)) * cos(radians(latitude)) *
			cos(radians(longitude) - radians(?)) +
			sin(radians(?)) * sin(radians(latitude))
		) <= 3
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

	if config.Get().IsProduction() {
		result, err := calculateDistances(coords, &shops)
		if err != nil {
			slog.Error("error al calcular distancias", "error", err)
		}

		resp = shops.ToDomainDTO(result)
	} else {
		dataDistance := make([]entity.ShopDistance, len(shops))
		for i, v := range shops {
			dataDistance[i].ShopID = v.ID
		}
		resp = shops.ToDomainDTO(dataDistance)
		for i := range resp {
			resp[i].DeliveryFee = deliveryfee.DevPlaceholderFee
			resp[i].Time = devTimeMinutes(resp[i].Type)
		}
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
		Slug:      shop.Slug,
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

// ListTurboCandidates retorna todos los nodos Turbo con licencia activa y
// coordenadas válidas, abiertos o no; la app decide cuál asignar.
func (s *shopsRepo) ListTurboCandidates(ctx context.Context) (shopsDTO.ShopsResponse, error) {
	var candidates shops.Shops

	if err := s.db.NewSelect().
		Model(&candidates).
		Where("type = ?", shops.ShopTypeTurbo).
		Where("license_status = ?", "active").
		Where("status = ?", shops.ShopStatusActive).
		Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Order("score DESC").
		Scan(ctx); err != nil {
		slog.Error("error al listar nodos turbo", "error", err)
		return shopsDTO.ShopsResponse{}, errcustom.ErrUnexpectedError
	}

	resp := make(shopsDTO.ShopsResponse, 0, len(candidates))
	for _, shop := range candidates {
		resp = append(resp, shopsDTO.ShopResponse{
			ID:        shop.ID,
			Name:      shop.Name,
			Slug:      shop.Slug,
			Address:   shop.Address,
			Latitude:  *shop.Latitude,
			Longitude: *shop.Longitude,
			HomePhone: shop.HomePhone,
			Logo:      shop.LogoImage,
			Banner:    shop.Banner,
			Opened:    shop.Opened,
			Type:      shop.Type,
			Score:     shop.Score,
		})
	}

	return resp, nil
}

// ResolveTurboShop calcula delivery_fee/time del nodo Turbo ya elegido para
// las coordenadas del cliente, igual que ToDomainDTO pero fuera del flujo de
// listado (production usa Google Maps; el resto, placeholders de dev).
func (s *shopsRepo) ResolveTurboShop(ctx context.Context, shopID uuid.UUID, coords shopsDTO.Coords) (shopsDTO.ShopResponse, error) {
	var shop shops.Shop
	if err := s.db.NewSelect().Model(&shop).Where("id = ?", shopID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shopsDTO.ShopResponse{}, errcustom.ErrNotFound
		}
		slog.Error("error al obtener nodo turbo", "error", err, "shopID", shopID)
		return shopsDTO.ShopResponse{}, errcustom.ErrUnexpectedError
	}

	distance := entity.ShopDistance{ShopID: shop.ID}
	if config.Get().IsProduction() {
		shopsToMeasure := shops.Shops{shop}
		result, err := calculateDistances(coords, &shopsToMeasure)
		if err != nil {
			slog.Error("error al calcular distancia del nodo turbo", "error", err, "shopID", shopID)
			return shopsDTO.ShopResponse{}, errcustom.ErrUnexpectedError
		}
		if len(result) > 0 {
			distance = result[0]
		}
	}

	resp := shop.ToDomainDTO(distance)
	if !config.Get().IsProduction() {
		resp.DeliveryFee = deliveryfee.DevPlaceholderFee
		resp.Time = devTimeMinutes(resp.Type)
	}

	return resp, nil
}

func (s *shopsRepo) SearchShopCandidates(ctx context.Context, query shopsDTO.SearchShopsQuery) ([]shopsDTO.SearchShopCandidate, error) {
	qNorm := strings.ToLower(strings.TrimSpace(query.Q))
	lat4 := strconv.FormatFloat(query.Latitude, 'f', 4, 64)
	lon4 := strconv.FormatFloat(query.Longitude, 'f', 4, 64)
	keyCache := "shops:search:" + qNorm + ":" + lat4 + ":" + lon4
	if query.Type != "" {
		keyCache += ":type:" + query.Type
	}
	cacheTTL := 2 * time.Minute

	if cached, err := s.cache.Get(ctx, keyCache); err == nil && cached != "" {
		var resp []shopsDTO.SearchShopCandidate
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return resp, nil
		}
		slog.Error("error al deserializar búsqueda del cache", "error", err)
	}

	radiusKm := coverageRadiusKm(query.Type)

	typeFilter := ""
	args := []any{
		qNorm, qNorm,
		query.Latitude, query.Longitude, query.Latitude,
		radiusKm,
		qNorm, qNorm, qNorm, qNorm, qNorm, qNorm,
	}
	if query.Type != "" {
		typeFilter = " AND s.type = ?"
		args = append(args, query.Type)
	}
	if query.Type == "" {
		// Sin type, la búsqueda general excluye Turbo (se descubre vía /turbo/entry).
		typeFilter = " AND s.type != ?"
		args = append(args, shops.ShopTypeTurbo)
	}

	rawSQL := `
		SELECT
			s.id, s.name, s.slug, s.address, s.latitude, s.longitude,
			s.home_phone, s.logo_image, s.banner, s.opened, s.type,
			s.score,
			(
				to_tsvector('spanish', s.name) @@ plainto_tsquery('spanish', ?)
				OR s.name ILIKE '%' || ? || '%'
			) AS matched_by_name
		FROM business.shops s
		WHERE s.opened = true
		  AND s.license_status = 'active'
		  AND s.status = 'active'
		  AND s.latitude IS NOT NULL
		  AND s.longitude IS NOT NULL
		  AND s.deleted_at IS NULL
		  AND 6371 * acos(
				cos(radians(?)) * cos(radians(s.latitude)) *
				cos(radians(s.longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(s.latitude))
			) <= ?
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
		  )` + typeFilter + `
		ORDER BY s.score DESC
	`

	var rows []shops.SearchShopRow
	if err := s.db.NewRaw(rawSQL, args...).Scan(ctx, &rows); err != nil {
		slog.Error("error en búsqueda de tiendas", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if len(rows) == 0 {
		return nil, nil
	}

	shopModels := make(shops.Shops, len(rows))
	for i, row := range rows {
		shopModels[i] = shops.Shop{
			ID:        row.ID,
			Name:      row.Name,
			Slug:      row.Slug,
			Address:   row.Address,
			HomePhone: row.HomePhone,
			LogoImage: row.LogoImage,
			Banner:    row.Banner,
			Opened:    row.Opened,
			Type:      row.Type,
			Score:     row.Score,
			Latitude:  row.Latitude,
			Longitude: row.Longitude,
		}
	}

	distances := make([]entity.ShopDistance, len(rows))
	for i, row := range rows {
		distances[i] = entity.ShopDistance{ShopID: row.ID}
	}

	if config.Get().IsProduction() {
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

	isProd := config.Get().IsProduction()
	candidates := make([]shopsDTO.SearchShopCandidate, 0, len(rows))
	for i, row := range rows {
		dist, ok := distMap[row.ID]
		if !ok {
			continue
		}
		shopResp := shopModels[i].ToDomainDTO(dist)
		if !isProd {
			shopResp.DeliveryFee = deliveryfee.DevPlaceholderFee
			shopResp.Time = devTimeMinutes(shopResp.Type)
		}
		candidates = append(candidates, shopsDTO.SearchShopCandidate{
			ShopResponse:  shopResp,
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
		Relation("Category").
		Relation("Category.Schedules").
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
		hasFlavors := item.HasFlavors || item.Details.HasFlavors()
		hasAdditions := item.Details.HasAdditions()

		var flavorSelection *shopsDTO.MatchedFlavorSelection
		if hasFlavors && item.Category != nil {
			flavorSelection = &shopsDTO.MatchedFlavorSelection{
				SelectorType: item.Category.Details.SelectorType,
				MinSelection: item.Details.MinSelection,
				MaxSelection: item.Details.MaxSelection,
			}
		}

		var additionSelection *shopsDTO.MatchedAdditionSelection
		if hasAdditions {
			additionSelection = &shopsDTO.MatchedAdditionSelection{
				SelectorType: productModels.SelectorMultiSelect,
				MinSelection: 0,
				MaxSelection: productModels.MaxAdditionsSelection,
			}
		}

		var details *shopsDTO.MatchedItemDetails
		if hasFlavors || hasAdditions {
			customerDetails := item.Details.ToCustomerDTO()
			matched := shopsDTO.MatchedItemDetails{
				MinSelection:          customerDetails.MinSelection,
				MaxSelection:          customerDetails.MaxSelection,
				MaxAdditionsSelection: customerDetails.MaxAdditionsSelection,
			}
			if len(customerDetails.Flavors) > 0 {
				matched.Flavors = make([]shopsDTO.MatchedFlavor, len(customerDetails.Flavors))
				for i, f := range customerDetails.Flavors {
					matched.Flavors[i] = shopsDTO.MatchedFlavor{
						ID:        f.ID,
						Name:      f.Name,
						Available: f.Available,
					}
				}
			}
			if len(customerDetails.Additions) > 0 {
				matched.Additions = make([]shopsDTO.MatchedAddition, len(customerDetails.Additions))
				for i, a := range customerDetails.Additions {
					matched.Additions[i] = shopsDTO.MatchedAddition{
						ID:        a.ID,
						Name:      a.Name,
						Ref:       a.Ref,
						Available: a.Available,
					}
				}
			}
			details = &matched
		}

		var schedules []shopsDTO.MatchedSchedule
		if item.Category != nil {
			schedules = make([]shopsDTO.MatchedSchedule, 0, len(item.Category.Schedules))
			for _, sch := range item.Category.Schedules {
				schedules = append(schedules, shopsDTO.MatchedSchedule{
					DayOfWeek: sch.DayOfWeek,
					StartTime: sch.StartTime,
					EndTime:   sch.EndTime,
					Enabled:   sch.Enabled,
				})
			}
		}

		result[item.ShopID] = append(result[item.ShopID], shopsDTO.MatchedProductItem{
			ID:                item.ID,
			ShopID:            item.ShopID,
			Name:              item.Name,
			Ref:               float64(item.Price),
			Image:             item.Image,
			Description:       item.Description,
			Score:             item.Score,
			HasFlavors:        hasFlavors,
			HasAdditions:      hasAdditions,
			FlavorSelection:   flavorSelection,
			AdditionSelection: additionSelection,
			Details:           details,
			Schedules:         schedules,
		})
	}

	return result, nil
}

func (s *shopsRepo) GetShopStatus(ctx context.Context, shopID uuid.UUID) (string, error) {
	var status string
	err := s.db.NewSelect().
		TableExpr("business.shops").
		ColumnExpr("status").
		Where("id = ?", shopID).
		Where("deleted_at IS NULL").
		Scan(ctx, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errcustom.ErrNotFound
		}
		slog.Error("error al obtener status del shop", "error", err, "shopID", shopID)
		return "", errcustom.ErrUnexpectedError
	}
	return status, nil
}

func (s *shopsRepo) SetOpenClose(ctx context.Context, shopID *uuid.UUID, open bool) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().
			TableExpr("business.shops").
			Set("opened = ?", open).
			Set("updated_at = now()").
			Where("id = ?", shopID)

		if open {
			// Si es la primera vez que el negocio abre, ancla las dos ventanas de
			// lanzamiento (día 1 al 50% y comisión 0% por 8 días) al instante
			// exacto de esta apertura. El CASE WHEN ... IS NULL evita que
			// aperturas posteriores reinicien las ventanas, sin necesitar un
			// SELECT ... FOR UPDATE aparte.
			now := time.Now()
			launchPromoEndsAt := now.Add(launchPromoWindow)
			zeroCommissionEndsAt := now.Add(zeroCommissionWindow)

			q = q.
				Set("first_opened_at = CASE WHEN first_opened_at IS NULL THEN ? ELSE first_opened_at END", now).
				Set("launch_promo_ends_at = CASE WHEN launch_promo_ends_at IS NULL THEN ? ELSE launch_promo_ends_at END", launchPromoEndsAt).
				Set("zero_commission_ends_at = CASE WHEN zero_commission_ends_at IS NULL THEN ? ELSE zero_commission_ends_at END", zeroCommissionEndsAt)
		}

		if _, err := q.Exec(ctx); err != nil {
			slog.Error("error al actualizar estado del local", "error", err, "shopID", shopID)
			return errcustom.ErrUnexpectedError
		}

		if open {
			if err := ensureOpenBatch(ctx, tx, *shopID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// ensureOpenBatch crea un lote abierto si el shop no tiene uno. Si ya existe
// (reapertura con jornada viva), no hace nada.
func ensureOpenBatch(ctx context.Context, tx bun.Tx, shopID uuid.UUID) error {
	var existing uuid.UUID
	err := tx.NewSelect().
		TableExpr("business.batches").
		Column("id").
		Where("shop_id = ? AND opened = ?", shopID, true).
		Limit(1).
		Scan(ctx, &existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		slog.Error("error al buscar lote abierto", "error", err, "shopID", shopID)
		return errcustom.ErrUnexpectedError
	}

	var lastNum int
	if err := tx.NewSelect().
		TableExpr("business.batches").
		ColumnExpr("COALESCE(MAX(batch), 0)").
		Where("shop_id = ?", shopID).
		Scan(ctx, &lastNum); err != nil {
		slog.Error("error al obtener número de lote", "error", err, "shopID", shopID)
		return errcustom.ErrUnexpectedError
	}

	batch := &shopsModels.Batch{
		ID:        uuid.New(),
		ShopID:    shopID,
		Batch:     lastNum + 1,
		Opened:    true,
		CreatedAt: time.Now(),
	}
	if _, err := tx.NewInsert().Model(batch).Exec(ctx); err != nil {
		slog.Error("error al crear lote", "error", err, "shopID", shopID)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (s *shopsRepo) GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error) {
	var endsAt *time.Time
	err := s.db.NewSelect().
		TableExpr("business.shops").
		Column("launch_promo_ends_at").
		Where("id = ?", shopID).
		Scan(ctx, &endsAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("error al obtener la ventana de promo de lanzamiento", "error", err, "shopID", shopID)
		return nil, errcustom.ErrUnexpectedError
	}
	return endsAt, nil
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

func (s *shopsRepo) TryCloseOpenBatch(ctx context.Context, shopID uuid.UUID) error {
	_, err := s.db.NewUpdate().
		TableExpr("business.batches AS b").
		Set("opened = ?", false).
		Set("close_at = now()").
		Where("b.shop_id = ?", shopID).
		Where("b.opened = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM business.shops s WHERE s.id = b.shop_id AND s.opened = true)").
		Where("NOT EXISTS (SELECT 1 FROM orders.orders o WHERE o.batch_id = b.id AND o.status IN (?))", bun.In(shopsModels.BatchBlockingOrderStatuses)).
		Exec(ctx)
	if err != nil {
		slog.Error("error al cerrar lote", "error", err, "shopID", shopID)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (s *shopsRepo) TryCloseAllEligibleBatches(ctx context.Context) error {
	_, err := s.db.NewUpdate().
		TableExpr("business.batches AS b").
		Set("opened = ?", false).
		Set("close_at = now()").
		Where("b.opened = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM business.shops s WHERE s.id = b.shop_id AND s.opened = true)").
		Where("NOT EXISTS (SELECT 1 FROM orders.orders o WHERE o.batch_id = b.id AND o.status IN (?))", bun.In(shopsModels.BatchBlockingOrderStatuses)).
		Exec(ctx)
	if err != nil {
		slog.Error("error al cerrar lotes elegibles", "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func calculateDistances(coords shopsDTO.Coords, shopsResult *shops.Shops) ([]entity.ShopDistance, error) {

	var shopIDs []uuid.UUID
	var destinations [][2]float64

	for _, shop := range *shopsResult {
		if shop.Latitude != nil && shop.Longitude != nil {
			shopIDs = append(shopIDs, shop.ID)
			destinations = append(destinations, [2]float64{*shop.Latitude, *shop.Longitude})
		}
	}

	if len(destinations) == 0 {
		return nil, errors.New("no shops with valid coordinates")
	}

	elements, err := googlemaps.DistanceMatrix(coords.Latitude, coords.Longitude, destinations)
	if err != nil {
		return nil, err
	}

	result := make([]entity.ShopDistance, 0, len(elements))
	for _, elem := range elements {
		result = append(result, entity.ShopDistance{
			ShopID:   shopIDs[elem.Index],
			Distance: elem.DistanceMeters,
			Duration: elem.DurationMin,
		})
	}

	return result, nil
}
