package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	productPorts "Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

const (
	prefix_shop_cashe         = "products:shop:"
	prefix_best_sellers_cache = "products:best-sellers:"
	prefix_additions_cache    = "products:additions:"
	prefix_flavors_item_cache = "products:flavors:item:"
	prefix_flavors_cat_cache  = "products:flavors:cat:"
	prefix_macro_categories   = "products:macro-categories"
)

type productsRepo struct {
	db    *bun.DB
	cache cache.Cache
}

// activeDiscountJoinConditions filtra la relación Item.Discount al único
// descuento activo y vigente del producto (el índice único parcial garantiza
// que a lo sumo una fila cumple is_active = true AND deleted_at IS NULL por
// item).
//
// El alias del JOIN generado por bun para una relación se deriva del nombre
// del campo Go de la relación (Discount -> discount), no del nombre de la
// tabla ni del tipo del modelo (item_discounts / ItemDiscount).
//
// IMPORTANTE: esta condición debe ir en el ON del LEFT JOIN vía
// RelationWithOpts/AdditionalJoinOnConditions, NUNCA como Relation(name, func(sq)
// sq.Where(...)). Un callback con .Where() en una relación has-one/belongs-to
// agrega la condición al WHERE de la consulta (no al ON), y como NULL = true
// es falso en SQL, eso convierte el LEFT JOIN en un INNER JOIN de facto: todo
// item SIN descuento activo (la mayoría) queda excluido del resultado entero,
// no solo su descuento.
func activeDiscountJoinConditions() []schema.QueryWithArgs {
	return []schema.QueryWithArgs{
		bun.SafeQuery("discount.is_active = ? AND discount.deleted_at IS NULL", true),
	}
}

func NewProductsRepo(db *bun.DB, cache cache.Cache) productPorts.ProductsRepo {
	return &productsRepo{db, cache}
}

func (p *productsRepo) RegisterItem(ctx context.Context, item *models.Item) error {
	item.ID = id.Ensure(item.ID)
	_, err := p.db.NewInsert().Model(item).Exec(ctx)
	if err != nil {
		slog.Error("Error al registrar producto", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearItemCaches(ctx, item.ShopID, item.ID)

	return err
}

func (p *productsRepo) UpdateItem(ctx context.Context, item *models.UpdateItem, shopID uuid.UUID) error {
	_, err := p.db.NewUpdate().Model(item).WherePK().Exec(ctx)
	if err != nil {
		slog.Error("Error al actualizar", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearItemCaches(ctx, shopID, item.ID)

	return err
}

func (p *productsRepo) SearchItemByID(ctx context.Context, id uuid.UUID) (*models.Item, error) {

	var item models.Item

	if err := p.db.NewSelect().
		Model(&item).
		Where("id = ?", id).
		Scan(ctx); err != nil {
		slog.Error("error al obtener el producto", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &item, nil
}

func (p *productsRepo) RegisterCategory(ctx context.Context, category *models.Category) error {
	category.ID = id.Ensure(category.ID)
	_, err := p.db.NewInsert().Model(category).Exec(ctx)
	if err != nil {
		slog.Error("error al registrar categoría", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+category.ShopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	return nil
}

func (p *productsRepo) SearchCategoryByID(ctx context.Context, categoryID uuid.UUID) (*models.Category, error) {
	var category models.Category

	if err := p.db.NewSelect().
		Model(&category).
		Where("id = ?", categoryID).
		Scan(ctx); err != nil {
		slog.Error("error al obtener la categoría", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &category, nil
}

func (p *productsRepo) UpdateCategory(ctx context.Context, category *models.UpdateCategory, shopID uuid.UUID) error {
	_, err := p.db.NewUpdate().Model(category).WherePK().Exec(ctx)
	if err != nil {
		slog.Error("error al actualizar la categoría", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	return nil
}

func (p *productsRepo) ReplaceCategorySchedules(ctx context.Context, categoryID, shopID uuid.UUID, schedules []models.Schedule) error {
	err := p.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*models.Schedule)(nil)).
			Where("category_id = ?", categoryID).
			Exec(ctx); err != nil {
			return err
		}

		if len(schedules) == 0 {
			return nil
		}
		for i := range schedules {
			schedules[i].ID = id.Ensure(schedules[i].ID)
		}

		_, err := tx.NewInsert().Model(&schedules).Exec(ctx)
		return err
	})
	if err != nil {
		slog.Error("error al reemplazar el horario de la categoría", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	return nil
}

func (p *productsRepo) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID, forCustomer bool) (dtos.CategoriesResponse, error) {

	cacheKey := prefix_shop_cashe + shopID.String()
	if forCustomer {
		cacheKey += ":customer"
	}

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.CategoriesResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de los productos", "key", cacheKey)
	}

	q := p.db.NewSelect().
		Model((*models.Category)(nil)).
		Where("shop_id = ?", shopID)

	if forCustomer {
		q = q.Where("availability_status != ?", models.AvailabilityHidden)
	}

	total, err := q.Count(ctx)
	if err != nil {
		slog.Error("error al contar categorías", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if total == 0 {
		return nil, errcustom.ErrNotFound
	}

	var categories models.Categories

	categoryQuery := p.db.NewSelect().
		Model(&categories).
		Where("shop_id = ?", shopID)

	if forCustomer {
		categoryQuery = categoryQuery.Where("availability_status != ?", models.AvailabilityHidden)
	}

	if err := categoryQuery.
		Relation("Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
			if forCustomer {
				return sq.Where("availability_status IN (?)", bun.In([]models.AvailabilityStatus{
					models.AvailabilityActive,
					models.AvailabilityOutOfStock,
				})).Order("price ASC")
			}
			return sq.Order("price ASC")
		}).
		RelationWithOpts("Items.Discount", bun.RelationOpts{
			AdditionalJoinOnConditions: activeDiscountJoinConditions(),
		}).
		Relation("Schedules").
		Relation("MacroCategory").
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	var data dtos.CategoriesResponse
	if forCustomer {
		data = categories.ToCustomerDTO()
	} else {
		data = categories.ToDomainDTO()
	}

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los productos en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) SearchAdditionsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.AdditionsResponse, error) {
	cacheKey := prefix_additions_cache + "item:" + itemID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.AdditionsResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de las adiciones del item", "key", cacheKey)
	}

	var item models.Item
	if err := p.db.NewSelect().
		Model(&item).
		Where("id = ?", itemID).
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("error al obtener el item de adiciones", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if !item.Details.HasAdditions() {
		return nil, errcustom.ErrNotFound
	}

	data := item.Details.Additions.ToCustomerDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar las adiciones del item en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) SearchAdditionsCatalogByShopID(ctx context.Context, shopID uuid.UUID) (dtos.AdditionsResponse, error) {
	cacheKey := prefix_additions_cache + "catalog:shop:" + shopID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.AdditionsResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal del catálogo de adiciones", "key", cacheKey)
	}

	var items models.Items
	if err := p.db.NewSelect().
		Model(&items).
		Join("JOIN products.categories AS category ON category.id = item.category_id").
		Join("JOIN products.macro_categories AS macro_category ON macro_category.id = category.macro_category_id").
		Where("item.shop_id = ?", shopID).
		Where("item.deleted_at IS NULL").
		Where("macro_category.slug = ?", models.MacroSlugAdiciones).
		Order("item.name ASC").
		Scan(ctx); err != nil {
		slog.Error("error al obtener catálogo de adiciones", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := make(dtos.AdditionsResponse, 0, len(items))
	for _, item := range items {
		entry := models.AdditionEntryFromItem(item)
		data = append(data, dtos.AdditionResponse{
			ID:        entry.ID,
			Name:      entry.Name,
			Ref:       entry.Price,
			Available: entry.IsAvailable(),
		})
	}

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar el catálogo de adiciones en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) SearchAdditionItemsByIDs(ctx context.Context, shopID uuid.UUID, additionIDs []uuid.UUID) (models.Items, error) {
	if len(additionIDs) == 0 {
		return nil, nil
	}

	var items models.Items
	if err := p.db.NewSelect().
		Model(&items).
		Join("JOIN products.categories AS category ON category.id = item.category_id").
		Join("JOIN products.macro_categories AS macro_category ON macro_category.id = category.macro_category_id").
		Where("item.shop_id = ?", shopID).
		Where("item.id IN (?)", bun.In(additionIDs)).
		Where("item.deleted_at IS NULL").
		Where("macro_category.slug = ?", models.MacroSlugAdiciones).
		Scan(ctx); err != nil {
		slog.Error("error al obtener items de adiciones por ids", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return items, nil
}

func (p *productsRepo) IsAdditionItem(ctx context.Context, itemID uuid.UUID) (bool, error) {
	count, err := p.db.NewSelect().
		Model((*models.Item)(nil)).
		Join("JOIN products.categories AS category ON category.id = item.category_id").
		Join("JOIN products.macro_categories AS macro_category ON macro_category.id = category.macro_category_id").
		Where("item.id = ?", itemID).
		Where("macro_category.slug = ?", models.MacroSlugAdiciones).
		Count(ctx)
	if err != nil {
		slog.Error("error al verificar si el item es adición", "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	return count > 0, nil
}

func (p *productsRepo) SyncAdditionSnapshots(ctx context.Context, shopID uuid.UUID, entry models.AdditionEntry) error {
	var items []models.Item
	if err := p.db.NewSelect().
		Model(&items).
		Column("id", "details").
		Where("shop_id = ?", shopID).
		Where("EXISTS (SELECT 1 FROM jsonb_array_elements(details->'additions') elem WHERE (elem->>'id')::uuid = ?)", entry.ID).
		Scan(ctx); err != nil {
		slog.Error("error al buscar items con la adición", "error", err)
		return errcustom.ErrUnexpectedError
	}

	itemIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		details := item.Details
		details.Additions = details.Additions.Upsert(entry)
		if _, err := p.db.NewUpdate().
			Model((*models.Item)(nil)).
			Set("details = ?", details).
			Where("id = ?", item.ID).
			Exec(ctx); err != nil {
			slog.Error("error al sincronizar snapshot de adición", "error", err)
			return errcustom.ErrUnexpectedError
		}
		itemIDs = append(itemIDs, item.ID)
	}

	p.clearAdditionCaches(ctx, shopID, itemIDs)
	return nil
}

func (p *productsRepo) SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error) {

	cacheKey := prefix_flavors_item_cache + itemID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.FlavorsResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de los sabores por item", "key", cacheKey)
	}

	item, err := p.SearchItemByID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if len(item.Details.Flavors) == 0 {
		return nil, errcustom.ErrNotFound
	}

	data := item.Details.Flavors.ToCustomerDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los sabores por item en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) SearchFlavorsByCategoryID(ctx context.Context, categoryID uuid.UUID) (dtos.FlavorsResponse, error) {

	cacheKey := prefix_flavors_cat_cache + categoryID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.FlavorsResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de los sabores por categoría", "key", cacheKey)
	}

	category, err := p.SearchCategoryByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	if !models.SupportsFlavorSelection(category.Details.SelectorType) {
		return nil, errcustom.ErrCategoryWithoutFlavors
	}

	if len(category.Details.Flavors) == 0 {
		return nil, errcustom.ErrNotFound
	}

	data := category.Details.Flavors.ToDomainDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los sabores por categoría en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error) {

	cacheKey := prefix_best_sellers_cache + shopID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.ItemsResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de los productos más vendidos", "key", cacheKey)
	}

	total, err := p.db.NewSelect().
		Model((*models.Item)(nil)).
		Where("shop_id = ?", shopID).
		Count(ctx)
	if err != nil {
		slog.Error("error al contar productos más vendidos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if total == 0 {
		return nil, errcustom.ErrNotFound
	}

	var topSeller models.Items

	if err := p.db.NewSelect().
		Model(&topSeller).
		// Calificadas con "item." porque el LEFT JOIN a item_discounts (alias
		// "discount") también tiene columnas "shop_id"; sin calificar,
		// Postgres reporta "column reference is ambiguous" (SQLSTATE=42702).
		Where("item.shop_id = ?", shopID).
		Where("item.is_available = ?", true).
		Relation("Category").
		Relation("Category.Schedules").
		RelationWithOpts("Discount", bun.RelationOpts{
			AdditionalJoinOnConditions: activeDiscountJoinConditions(),
		}).
		Order("sold desc").
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos mas vendidos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := topSeller.ToDomainDTOWithCategory()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los productos mas vendidos en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) AddSoldItem(ctx context.Context, item models.DataProduceOrder) error {

	_, err := p.db.NewUpdate().
		Model(&item).
		Set("sold = sold + ?", item.Quantity).
		Where("id = ?", item.ProductID).
		Exec(ctx)
	return err

}

const prefix_trending_cache = "products:trending:"

// trendingCacheBundle es lo que realmente se cachea 15 minutos: la fila cruda
// de BD más los cálculos costosos (Google Maps, horarios). Deliberadamente NO
// incluye EffectiveRef/HasLaunchPromo: esos se recalculan en cada request
// (hit o miss) a partir de ShopLaunchPromoEndsAt, para que el badge de la
// promo día 1 nunca quede desfasado más de lo que tarda una comparación de
// timestamps en memoria.
type trendingCacheBundle struct {
	Rows      []models.TrendingRow
	Fee       map[uuid.UUID]float64
	Minutes   map[uuid.UUID]int
	Schedules map[uuid.UUID][]dtos.ScheduleResponse
}

func (p *productsRepo) GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error) {
	lat, lon := coords.Latitude, coords.Longitude
	cacheKey := fmt.Sprintf("%s%v_%v", prefix_trending_cache, lat, lon)

	if cached, err := p.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var bundle trendingCacheBundle
		if err := json.Unmarshal([]byte(cached), &bundle); err == nil {
			return buildTrendingResponse(bundle), nil
		}
		slog.Error("error al deserializar trending del cache", "key", cacheKey)
	}

	var rows []models.TrendingRow

	query, args := TrendingQuery(config.Get().IsProduction(), lat, lon)
	err := p.db.NewRaw(query, args...).Scan(ctx, &rows)
	if err != nil {
		slog.Error("error al obtener trending products", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	fee, minutes := p.deliveryEstimatesByShop(coords, rows)
	bundle := trendingCacheBundle{
		Rows:      rows,
		Fee:       fee,
		Minutes:   minutes,
		Schedules: p.schedulesByCategory(ctx, rows),
	}

	if err := p.cache.Set(ctx, cacheKey, bundle, 15*time.Minute); err != nil {
		slog.Error("error al guardar trending en cache", "error", err)
	}

	return buildTrendingResponse(bundle), nil
}

// TrendingQuery usa el mismo radio de cobertura que el listado de negocios para
// que trending nunca quede vacío mientras /shops sí muestra negocios.
func TrendingQuery(isProduction bool, lat, lon float64) (string, []any) {
	query := `
		SELECT
			s.id::text        AS shop_id,
			s.name            AS shop_name,
			s.slug            AS shop_slug,
			s.logo_image      AS shop_logo,
			s.address         AS shop_address,
			s.score           AS shop_score,
			s.type            AS shop_type,
			s.latitude        AS shop_latitude,
			s.longitude       AS shop_longitude,
			s.launch_promo_ends_at AS shop_launch_promo_ends_at,
			i.id::text        AS item_id,
			i.category_id::text AS item_category_id,
			i.name            AS item_name,
			i.price           AS item_price,
			i.image           AS item_image,
			i.description     AS item_desc,
			i.score           AS item_score,
			i.sold            AS item_sold
		FROM business.shops s
		JOIN products.items i ON i.shop_id = s.id
		WHERE s.opened = true
			AND s.license_status = 'active'
			AND i.is_available = true
			AND i.availability_status != ?
			AND i.sold > 0
			AND s.type != ?
	`
	args := []any{models.AvailabilityHidden, shops.ShopTypeTurbo}
	if isProduction {
		query += `
			AND s.latitude IS NOT NULL
			AND s.longitude IS NOT NULL
			AND 6371 * acos(
				cos(radians(?)) * cos(radians(s.latitude)) *
				cos(radians(s.longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(s.latitude))
			) <= ?
		`
		args = append(args, lat, lon, lat, shops.CoverageRadiusKm)
	}
	query += `
		ORDER BY i.sold DESC
		LIMIT 30
	`
	return query, args
}

// buildTrendingResponse arma la respuesta pública a partir del bundle
// (cacheado o recién consultado), calculando EffectiveRef/HasLaunchPromo en
// vivo contra la hora actual en vez de contra el momento en que se cacheó.
func buildTrendingResponse(bundle trendingCacheBundle) dtos.TrendingResponse {
	data := make(dtos.TrendingResponse, 0, len(bundle.Rows))
	for _, r := range bundle.Rows {
		shopID, _ := uuid.Parse(r.ShopID)
		itemID, _ := uuid.Parse(r.ItemID)
		categoryID, _ := uuid.Parse(r.ItemCategoryID)
		ref := r.ItemPrice
		effectiveRef := ref
		hasLaunchPromo := r.HasActiveLaunchPromo()
		if hasLaunchPromo {
			effectiveRef = money.RoundBs(ref * 0.5)
		}

		data = append(data, dtos.TrendingEntry{
			Shop: dtos.TrendingShop{
				ID:          shopID,
				Name:        r.ShopName,
				Slug:        r.ShopSlug,
				Logo:        r.ShopLogo,
				Address:     r.ShopAddress,
				Score:       r.ShopScore,
				Type:        r.ShopType,
				DeliveryFee: bundle.Fee[shopID],
				Time:        bundle.Minutes[shopID],
			},
			Product: dtos.TrendingProduct{
				ID:             itemID,
				CategoryID:     categoryID,
				Name:           r.ItemName,
				Ref:            ref,
				EffectiveRef:   effectiveRef,
				HasLaunchPromo: hasLaunchPromo,
				Image:          r.ItemImage,
				Description:    r.ItemDesc,
				Score:          r.ItemScore,
				Sold:           r.ItemSold,
				Schedules:      bundle.Schedules[categoryID],
			},
		})
	}
	return data
}

// schedulesByCategory carga en una sola query los horarios de todas las categorías
// presentes en rows, agrupados por category_id, para evitar N+1 queries.
func (p *productsRepo) schedulesByCategory(ctx context.Context, rows []models.TrendingRow) map[uuid.UUID][]dtos.ScheduleResponse {
	result := make(map[uuid.UUID][]dtos.ScheduleResponse)

	categoryIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{})
	for _, r := range rows {
		categoryID, err := uuid.Parse(r.ItemCategoryID)
		if err != nil {
			continue
		}
		if _, ok := seen[categoryID]; ok {
			continue
		}
		seen[categoryID] = struct{}{}
		categoryIDs = append(categoryIDs, categoryID)
	}

	if len(categoryIDs) == 0 {
		return result
	}

	var schedules models.Schedules
	if err := p.db.NewSelect().
		Model(&schedules).
		Where("category_id IN (?)", bun.In(categoryIDs)).
		Scan(ctx); err != nil {
		slog.Error("error al obtener horarios para trending", "error", err)
		return result
	}

	for _, s := range schedules {
		if s.CategoryID == nil {
			continue
		}
		result[*s.CategoryID] = append(result[*s.CategoryID], s.ToDomainDTO())
	}

	return result
}

// deliveryEstimatesByShop returns delivery fee (USD) and estimated minutes
// per distinct shop_id in rows, mirroring shops.Shop.ToDomainDTO's formula
// (delivery fee from distance, time = driving minutes + 20 prep minutes).
// Outside production it returns the same fixed placeholder shops/repo uses,
// to avoid depending on the Google Maps API in dev/test.
func (p *productsRepo) deliveryEstimatesByShop(coords dtos.TrendingCoords, rows []models.TrendingRow) (map[uuid.UUID]float64, map[uuid.UUID]int) {
	fee := make(map[uuid.UUID]float64)
	minutes := make(map[uuid.UUID]int)

	if !config.Get().IsProduction() {
		for _, r := range rows {
			shopID, err := uuid.Parse(r.ShopID)
			if err != nil {
				continue
			}
			fee[shopID] = deliveryfee.DevPlaceholderFee
			minutes[shopID] = 12
		}
		return fee, minutes
	}

	var shopIDs []uuid.UUID
	var destinations [][2]float64
	seen := make(map[uuid.UUID]struct{})

	for _, r := range rows {
		shopID, err := uuid.Parse(r.ShopID)
		if err != nil {
			continue
		}
		if _, ok := seen[shopID]; ok {
			continue
		}
		seen[shopID] = struct{}{}
		shopIDs = append(shopIDs, shopID)
		destinations = append(destinations, [2]float64{r.ShopLatitude, r.ShopLongitude})
	}

	if len(destinations) == 0 {
		return fee, minutes
	}

	elements, err := googlemaps.DistanceMatrix(coords.Latitude, coords.Longitude, destinations)
	if err != nil {
		slog.Error("error al calcular distancias para trending", "error", err)
		return fee, minutes
	}

	for _, elem := range elements {
		shopID := shopIDs[elem.Index]
		fee[shopID] = deliveryfee.FromDistance(elem.DistanceMeters)
		minutes[shopID] = int(elem.DurationMin) + 20
	}

	return fee, minutes
}

const prefix_supermarket_home_cache = "products:supermarket-home:"

// supermarketHomeMaxPerShopPerShelf acota, en SQL, cuántos productos de un
// mismo negocio pueden entrar en una repisa (micro-categoría) — sin esto, un
// super con catálogo grande eclipsaría a los demás negocios de la zona.
const supermarketHomeMaxPerShopPerShelf = 4

// supermarketHomeCacheBundle es lo que se cachea 15 minutos: filas crudas más
// los cálculos costosos (Google Maps, horarios), igual que trending.
type supermarketHomeCacheBundle struct {
	Rows      []models.SupermarketHomeRow
	Fee       map[uuid.UUID]float64
	Minutes   map[uuid.UUID]int
	Schedules map[uuid.UUID][]dtos.ScheduleResponse
}

// SearchSupermarketCategoryPool retorna las (macro, categoría) distintas
// disponibles hoy en la zona para supermercados — es el universo del que el
// app layer elige la rotación diaria. Consulta liviana, sin cache.
func (p *productsRepo) SearchSupermarketCategoryPool(ctx context.Context, coords dtos.SupermarketHomeCoords) ([]models.SupermarketCategoryCandidate, error) {
	var pool []models.SupermarketCategoryCandidate
	query, args := SupermarketCategoryPoolQuery(config.Get().IsProduction(), coords.Latitude, coords.Longitude)
	if err := p.db.NewRaw(query, args...).Scan(ctx, &pool); err != nil {
		slog.Error("error al obtener el pool de categorías de supermarket home", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return pool, nil
}

// SupermarketCategoryPoolQuery usa el mismo radio de cobertura que el listado
// de negocios (CoverageRadiusKm), restringido a shops type=Supermarket.
func SupermarketCategoryPoolQuery(isProduction bool, lat, lon float64) (string, []any) {
	query := `
		SELECT DISTINCT
			mc.id::text AS macro_id,
			mc.name     AS macro_name,
			mc.slug     AS macro_slug,
			c.name      AS category_name
		FROM business.shops s
		JOIN products.categories c ON c.shop_id = s.id
		JOIN products.macro_categories mc ON mc.id = c.macro_category_id
		JOIN products.items i ON i.category_id = c.id
		WHERE s.opened = true
			AND s.license_status = 'active'
			AND s.type = ?
			AND c.availability_status != ?
			AND i.is_available = true
			AND i.availability_status != ?
	`
	args := []any{shops.ShopTypeSupermarket, models.AvailabilityHidden, models.AvailabilityHidden}
	if isProduction {
		query += `
			AND s.latitude IS NOT NULL
			AND s.longitude IS NOT NULL
			AND 6371 * acos(
				cos(radians(?)) * cos(radians(s.latitude)) *
				cos(radians(s.longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(s.latitude))
			) <= ?
		`
		args = append(args, lat, lon, lat, shops.CoverageRadiusKm)
	}
	return query, args
}

// SearchSupermarketHomeEntries retorna, para el conjunto de categorías ya
// elegido por la rotación diaria del app layer, los productos de negocios
// Supermarket de la zona (tope supermarketHomeMaxPerShopPerShelf por negocio
// por categoría), con delivery fee/tiempo y horarios ya resueltos. Cachea 15
// minutos; el key incluye la fecha de negocio porque la rotación cambia cada
// día.
func (p *productsRepo) SearchSupermarketHomeEntries(ctx context.Context, coords dtos.SupermarketHomeCoords, categoryNames []string) (dtos.HomeEntriesResponse, error) {
	if len(categoryNames) == 0 {
		return dtos.HomeEntriesResponse{}, nil
	}

	lat, lon := coords.Latitude, coords.Longitude
	dateKey := bizclock.Now().Format("2006-01-02")
	cacheKey := fmt.Sprintf("%s%v_%v_%s", prefix_supermarket_home_cache, lat, lon, dateKey)

	if cached, err := p.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var bundle supermarketHomeCacheBundle
		if err := json.Unmarshal([]byte(cached), &bundle); err == nil {
			return buildSupermarketHomeEntries(bundle), nil
		}
		slog.Error("error al deserializar supermarket home del cache", "key", cacheKey)
	}

	var rows []models.SupermarketHomeRow
	query, args := SupermarketHomeProductsQuery(config.Get().IsProduction(), lat, lon, categoryNames, supermarketHomeMaxPerShopPerShelf)
	if err := p.db.NewRaw(query, args...).Scan(ctx, &rows); err != nil {
		slog.Error("error al obtener productos de supermarket home", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	fee, minutes := p.deliveryEstimatesForHome(coords, rows)
	bundle := supermarketHomeCacheBundle{
		Rows:      rows,
		Fee:       fee,
		Minutes:   minutes,
		Schedules: p.schedulesByHomeCategory(ctx, rows),
	}

	if err := p.cache.Set(ctx, cacheKey, bundle, 15*time.Minute); err != nil {
		slog.Error("error al guardar supermarket home en cache", "error", err)
	}

	return buildSupermarketHomeEntries(bundle), nil
}

// SupermarketHomeProductsQuery rankea con ROW_NUMBER() por (categoría, shop)
// para acotar en SQL cuántos productos de un mismo negocio entran en cada
// repisa, antes de que el app layer filtre por horario y recorte al tope
// final por repisa.
func SupermarketHomeProductsQuery(isProduction bool, lat, lon float64, categoryNames []string, maxPerShopPerShelf int) (string, []any) {
	placeholders := make([]string, len(categoryNames))
	args := []any{shops.ShopTypeSupermarket, models.AvailabilityHidden, models.AvailabilityHidden}
	for i, name := range categoryNames {
		placeholders[i] = "?"
		args = append(args, name)
	}

	query := `
		WITH ranked AS (
			SELECT
				s.id::text        AS shop_id,
				s.name            AS shop_name,
				s.slug            AS shop_slug,
				s.logo_image      AS shop_logo,
				s.latitude        AS shop_latitude,
				s.longitude       AS shop_longitude,
				c.id::text        AS category_id,
				c.name            AS category_name,
				mc.id::text       AS macro_id,
				mc.name           AS macro_name,
				mc.slug           AS macro_slug,
				i.id::text        AS item_id,
				i.name            AS item_name,
				i.price           AS item_price,
				i.image           AS item_image,
				i.score           AS item_score,
				i.sold            AS item_sold,
				ROW_NUMBER() OVER (PARTITION BY c.name, s.id ORDER BY i.sold DESC) AS shop_rank
			FROM business.shops s
			JOIN products.categories c ON c.shop_id = s.id
			JOIN products.macro_categories mc ON mc.id = c.macro_category_id
			JOIN products.items i ON i.category_id = c.id
			WHERE s.opened = true
				AND s.license_status = 'active'
				AND s.type = ?
				AND c.availability_status != ?
				AND i.is_available = true
				AND i.availability_status != ?
				AND c.name IN (` + strings.Join(placeholders, ",") + `)
	`
	if isProduction {
		query += `
				AND s.latitude IS NOT NULL
				AND s.longitude IS NOT NULL
				AND 6371 * acos(
					cos(radians(?)) * cos(radians(s.latitude)) *
					cos(radians(s.longitude) - radians(?)) +
					sin(radians(?)) * sin(radians(s.latitude))
				) <= ?
		`
		args = append(args, lat, lon, lat, shops.CoverageRadiusKm)
	}
	query += `
		)
		SELECT shop_id, shop_name, shop_slug, shop_logo, shop_latitude, shop_longitude,
			category_id, category_name, macro_id, macro_name, macro_slug,
			item_id, item_name, item_price, item_image, item_score, item_sold
		FROM ranked
		WHERE shop_rank <= ?
		ORDER BY category_name, item_sold DESC
	`
	args = append(args, maxPerShopPerShelf)
	return query, args
}

// buildSupermarketHomeEntries arma la lista plana a partir del bundle
// (cacheado o recién consultado). El app layer la filtra en vivo por horario
// y la agrupa en macro -> micro -> productos.
func buildSupermarketHomeEntries(bundle supermarketHomeCacheBundle) dtos.HomeEntriesResponse {
	data := make(dtos.HomeEntriesResponse, 0, len(bundle.Rows))
	for _, r := range bundle.Rows {
		shopID, _ := uuid.Parse(r.ShopID)
		itemID, _ := uuid.Parse(r.ItemID)
		categoryID, _ := uuid.Parse(r.CategoryID)

		data = append(data, dtos.HomeEntry{
			CategoryName: r.CategoryName,
			Shop: dtos.HomeShop{
				ID:          shopID,
				Name:        r.ShopName,
				Slug:        r.ShopSlug,
				Logo:        r.ShopLogo,
				DeliveryFee: bundle.Fee[shopID],
				Time:        bundle.Minutes[shopID],
			},
			Product: dtos.HomeProduct{
				ID:    itemID,
				Name:  r.ItemName,
				Ref:   r.ItemPrice,
				Image: r.ItemImage,
				Score: r.ItemScore,
				Sold:  r.ItemSold,
			},
			Schedules: bundle.Schedules[categoryID],
		})
	}
	return data
}

// schedulesByHomeCategory carga en una sola query los horarios de todas las
// categorías presentes en rows, agrupados por category_id — mismo propósito
// que schedulesByCategory pero para SupermarketHomeRow.
func (p *productsRepo) schedulesByHomeCategory(ctx context.Context, rows []models.SupermarketHomeRow) map[uuid.UUID][]dtos.ScheduleResponse {
	result := make(map[uuid.UUID][]dtos.ScheduleResponse)

	categoryIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{})
	for _, r := range rows {
		categoryID, err := uuid.Parse(r.CategoryID)
		if err != nil {
			continue
		}
		if _, ok := seen[categoryID]; ok {
			continue
		}
		seen[categoryID] = struct{}{}
		categoryIDs = append(categoryIDs, categoryID)
	}

	if len(categoryIDs) == 0 {
		return result
	}

	var schedules models.Schedules
	if err := p.db.NewSelect().
		Model(&schedules).
		Where("category_id IN (?)", bun.In(categoryIDs)).
		Scan(ctx); err != nil {
		slog.Error("error al obtener horarios para supermarket home", "error", err)
		return result
	}

	for _, s := range schedules {
		if s.CategoryID == nil {
			continue
		}
		result[*s.CategoryID] = append(result[*s.CategoryID], s.ToDomainDTO())
	}

	return result
}

// deliveryEstimatesForHome es deliveryEstimatesByShop pero para
// SupermarketHomeRow — misma fórmula (fee por distancia, tiempo = minutos de
// viaje + 20 de preparación), mismo placeholder fuera de producción.
func (p *productsRepo) deliveryEstimatesForHome(coords dtos.SupermarketHomeCoords, rows []models.SupermarketHomeRow) (map[uuid.UUID]float64, map[uuid.UUID]int) {
	fee := make(map[uuid.UUID]float64)
	minutes := make(map[uuid.UUID]int)

	if !config.Get().IsProduction() {
		for _, r := range rows {
			shopID, err := uuid.Parse(r.ShopID)
			if err != nil {
				continue
			}
			fee[shopID] = deliveryfee.DevPlaceholderFee
			minutes[shopID] = 12
		}
		return fee, minutes
	}

	var shopIDs []uuid.UUID
	var destinations [][2]float64
	seen := make(map[uuid.UUID]struct{})

	for _, r := range rows {
		shopID, err := uuid.Parse(r.ShopID)
		if err != nil {
			continue
		}
		if _, ok := seen[shopID]; ok {
			continue
		}
		seen[shopID] = struct{}{}
		shopIDs = append(shopIDs, shopID)
		destinations = append(destinations, [2]float64{r.ShopLatitude, r.ShopLongitude})
	}

	if len(destinations) == 0 {
		return fee, minutes
	}

	elements, err := googlemaps.DistanceMatrix(coords.Latitude, coords.Longitude, destinations)
	if err != nil {
		slog.Error("error al calcular distancias para supermarket home", "error", err)
		return fee, minutes
	}

	for _, elem := range elements {
		shopID := shopIDs[elem.Index]
		fee[shopID] = deliveryfee.FromDistance(elem.DistanceMeters)
		minutes[shopID] = int(elem.DurationMin) + 20
	}

	return fee, minutes
}

func (p *productsRepo) RegisterFlavor(ctx context.Context, flavor models.CatalogFlavor) error {
	category, err := p.SearchCategoryByID(ctx, flavor.CategoryID)
	if err != nil {
		return err
	}

	category.Details.Flavors = append(category.Details.Flavors, flavor.Entry)

	if _, err := p.db.NewUpdate().
		Model(category).
		Column("details").
		WherePK().
		Exec(ctx); err != nil {
		slog.Error("error al registrar sabor", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_flavors_cat_cache+flavor.CategoryID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+flavor.ShopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	return nil
}

func (p *productsRepo) SearchFlavorByID(ctx context.Context, flavorID uuid.UUID) (*models.CatalogFlavor, error) {
	var category models.Category

	if err := p.db.NewSelect().
		Model(&category).
		Where("EXISTS (SELECT 1 FROM jsonb_array_elements(details->'flavors') elem WHERE (elem->>'id')::uuid = ?)", flavorID).
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("error al obtener el sabor", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	entry, ok := category.Details.Flavors.FindByID(flavorID)
	if !ok {
		return nil, errcustom.ErrNotFound
	}

	return &models.CatalogFlavor{
		ID:         flavorID,
		ShopID:     category.ShopID,
		CategoryID: category.ID,
		Entry:      entry,
	}, nil
}

func (p *productsRepo) UpdateFlavor(ctx context.Context, flavorID uuid.UUID, entry models.FlavorEntry, shopID, categoryID uuid.UUID) error {
	itemIDs, err := p.itemIDsByFlavor(ctx, categoryID, flavorID)
	if err != nil {
		return err
	}

	err = p.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		category, err := p.searchCategoryByIDTx(ctx, tx, categoryID)
		if err != nil {
			return err
		}

		category.Details.Flavors = category.Details.Flavors.Upsert(entry)
		if _, err := tx.NewUpdate().
			Model(category).
			Column("details").
			WherePK().
			Exec(ctx); err != nil {
			return err
		}

		return p.syncItemFlavorsTx(ctx, tx, categoryID, flavorID, entry)
	})
	if err != nil {
		slog.Error("error al actualizar el sabor", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearFlavorRelatedCaches(ctx, shopID, categoryID, itemIDs)
	return nil
}

func (p *productsRepo) DeleteFlavor(ctx context.Context, flavor models.CatalogFlavor) error {
	itemIDs, err := p.itemIDsByFlavor(ctx, flavor.CategoryID, flavor.ID)
	if err != nil {
		return err
	}

	err = p.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		category, err := p.searchCategoryByIDTx(ctx, tx, flavor.CategoryID)
		if err != nil {
			return err
		}

		category.Details.Flavors = category.Details.Flavors.RemoveByID(flavor.ID)
		if _, err := tx.NewUpdate().
			Model(category).
			Column("details").
			WherePK().
			Exec(ctx); err != nil {
			return err
		}

		var items []models.Item
		if err := tx.NewSelect().
			Model(&items).
			Column("id", "details").
			Where("category_id = ?", flavor.CategoryID).
			Where("EXISTS (SELECT 1 FROM jsonb_array_elements(details->'flavors') elem WHERE (elem->>'id')::uuid = ?)", flavor.ID).
			Scan(ctx); err != nil {
			return err
		}

		for _, item := range items {
			details := item.Details
			details.Flavors = details.Flavors.RemoveByID(flavor.ID)
			if _, err := tx.NewUpdate().
				Model((*models.Item)(nil)).
				Set("details = ?", details).
				Set("has_flavors = ?", details.HasFlavors()).
				Where("id = ?", item.ID).
				Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		slog.Error("error al eliminar el sabor", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearFlavorRelatedCaches(ctx, flavor.ShopID, flavor.CategoryID, itemIDs)
	return nil
}

func (p *productsRepo) searchCategoryByIDTx(ctx context.Context, tx bun.Tx, categoryID uuid.UUID) (*models.Category, error) {
	var category models.Category
	if err := tx.NewSelect().
		Model(&category).
		Where("id = ?", categoryID).
		Scan(ctx); err != nil {
		return nil, err
	}
	return &category, nil
}

func (p *productsRepo) syncItemFlavorsTx(ctx context.Context, tx bun.Tx, categoryID, flavorID uuid.UUID, entry models.FlavorEntry) error {
	var items []models.Item
	if err := tx.NewSelect().
		Model(&items).
		Column("id", "details", "availability_status").
		Where("category_id = ?", categoryID).
		Where("EXISTS (SELECT 1 FROM jsonb_array_elements(details->'flavors') elem WHERE (elem->>'id')::uuid = ?)", flavorID).
		Scan(ctx); err != nil {
		return err
	}

	for _, item := range items {
		details := item.Details
		details.Flavors = details.Flavors.Upsert(entry)
		effective := models.EffectiveItemAvailability(item.AvailabilityStatus, details)
		if _, err := tx.NewUpdate().
			Model((*models.Item)(nil)).
			Set("details = ?", details).
			Set("has_flavors = ?", details.HasFlavors()).
			Set("is_available = ?", models.SyncIsAvailableFlag(effective)).
			Where("id = ?", item.ID).
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

// itemIDsByFlavor devuelve los items que tienen asignado un sabor embebido.
func (p *productsRepo) itemIDsByFlavor(ctx context.Context, categoryID, flavorID uuid.UUID) ([]uuid.UUID, error) {
	var itemIDs []uuid.UUID

	if err := p.db.NewSelect().
		Model((*models.Item)(nil)).
		Column("id").
		Where("category_id = ?", categoryID).
		Where("EXISTS (SELECT 1 FROM jsonb_array_elements(details->'flavors') elem WHERE (elem->>'id')::uuid = ?)", flavorID).
		Scan(ctx, &itemIDs); err != nil {
		slog.Error("error al obtener items por sabor", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return itemIDs, nil
}

func (p *productsRepo) clearFlavorRelatedCaches(ctx context.Context, shopID, categoryID uuid.UUID, itemIDs []uuid.UUID) {
	if err := p.cache.ClearByPrefix(ctx, prefix_flavors_cat_cache+categoryID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	for _, itemID := range itemIDs {
		if err := p.cache.ClearByPrefix(ctx, prefix_flavors_item_cache+itemID.String()); err != nil {
			slog.Error("error al limpiar el cache", "error", err)
		}
	}
}

func (p *productsRepo) SearchMacroCategories(ctx context.Context, shopType string) (models.MacroCategories, error) {
	keyCache := prefix_macro_categories
	if shopType != "" {
		keyCache += ":" + shopType
	}

	cached, _ := p.cache.Get(ctx, keyCache)
	if cached != "" {
		var data models.MacroCategories
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de las macro categorías", "key", keyCache)
	}

	var macroCategories models.MacroCategories

	q := p.db.NewSelect().
		Model(&macroCategories).
		Order("name ASC")
	if shopType != "" {
		q = q.Where("shop_type = ?", shopType)
	}

	if err := q.Scan(ctx); err != nil {
		slog.Error("error al obtener las macro categorías", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if err := p.cache.Set(ctx, keyCache, macroCategories, 24*time.Hour); err != nil {
		slog.Error("error al guardar las macro categorías en el cache", "error", err)
	}

	return macroCategories, nil
}

func (p *productsRepo) SearchMicroCategories(ctx context.Context, shopType string) (models.MicroCategories, error) {
	var microCategories models.MicroCategories

	if err := p.db.NewSelect().
		Model(&microCategories).
		Where("macro_category_id IN (SELECT id FROM products.macro_categories WHERE shop_type = ?)", shopType).
		Order("name ASC").
		Scan(ctx); err != nil {
		slog.Error("error al obtener las micro categorías", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return microCategories, nil
}

func (p *productsRepo) RegisterItemDiscount(ctx context.Context, discount *models.ItemDiscount) error {
	discount.ID = id.Ensure(discount.ID)
	_, err := p.db.NewInsert().Model(discount).Exec(ctx)
	if err != nil {
		slog.Error("error al registrar el descuento del producto", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearItemDiscountCaches(ctx, discount.ShopID)

	return nil
}

func (p *productsRepo) SearchItemDiscountByID(ctx context.Context, discountID uuid.UUID) (*models.ItemDiscount, error) {
	var discount models.ItemDiscount

	if err := p.db.NewSelect().
		Model(&discount).
		Where("id = ?", discountID).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		slog.Error("error al obtener el descuento del producto", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &discount, nil
}

func (p *productsRepo) SearchActiveItemDiscountByItemID(ctx context.Context, itemID uuid.UUID) (*models.ItemDiscount, error) {
	var discount models.ItemDiscount

	if err := p.db.NewSelect().
		Model(&discount).
		Where("item_id = ?", itemID).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("error al obtener el descuento activo del producto", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &discount, nil
}

func (p *productsRepo) UpdateItemDiscount(ctx context.Context, discount *models.UpdateItemDiscount, shopID, itemID uuid.UUID) error {
	_, err := p.db.NewUpdate().Model(discount).WherePK().Exec(ctx)
	if err != nil {
		slog.Error("error al actualizar el descuento del producto", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearItemDiscountCaches(ctx, shopID)

	return nil
}

func (p *productsRepo) DeleteItemDiscount(ctx context.Context, discount *models.ItemDiscount) error {
	_, err := p.db.NewUpdate().
		Model((*models.ItemDiscount)(nil)).
		Set("is_active = ?", false).
		Set("deleted_at = ?", time.Now()).
		Where("id = ?", discount.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("error al eliminar el descuento del producto", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearItemDiscountCaches(ctx, discount.ShopID)

	return nil
}

func (p *productsRepo) SearchCategoriesByIDs(ctx context.Context, shopID uuid.UUID, categoryIDs []uuid.UUID) (models.Categories, error) {
	var categories models.Categories

	if err := p.db.NewSelect().
		Model(&categories).
		Where("shop_id = ?", shopID).
		Where("id IN (?)", bun.In(categoryIDs)).
		Scan(ctx); err != nil {
		slog.Error("error al obtener categorías por ids", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return categories, nil
}

func (p *productsRepo) SearchCategoriesByShopID(ctx context.Context, shopID uuid.UUID) (models.Categories, error) {
	var categories models.Categories

	if err := p.db.NewSelect().
		Model(&categories).
		Where("shop_id = ?", shopID).
		Relation("Schedules").
		Relation("MacroCategory").
		Order("name ASC").
		Scan(ctx); err != nil {
		slog.Error("error al obtener categorías del negocio", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if len(categories) == 0 {
		return nil, errcustom.ErrNotFound
	}

	return categories, nil
}

func (p *productsRepo) SearchCategorySummariesByShopID(ctx context.Context, shopID uuid.UUID, filter string) (models.Categories, error) {
	var categories models.Categories

	q := p.db.NewSelect().
		Model(&categories).
		Column("id", "name").
		Where("shop_id = ?", shopID).
		Where("deleted_at IS NULL")

	switch filter {
	case "flavors":
		q = q.Where("details->>'selector_type' IN (?)", bun.In([]string{
			models.SelectorSingleSelect,
			models.SelectorMultiSelect,
		}))
	case "additions":
		q = q.
			Join("JOIN products.macro_categories AS macro_category ON macro_category.id = category.macro_category_id").
			Where("macro_category.slug = ?", models.MacroSlugAdiciones)
	}

	if err := q.Order("name ASC").Scan(ctx); err != nil {
		slog.Error("error al obtener el resumen de categorías", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return categories, nil
}

func (p *productsRepo) clearShopCache(ctx context.Context, shopID uuid.UUID) {
	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache del negocio", "error", err)
	}
}

func (p *productsRepo) clearItemCaches(ctx context.Context, shopID, itemID uuid.UUID) {
	p.clearShopCache(ctx, shopID)

	if err := p.cache.ClearByPrefix(ctx, prefix_flavors_item_cache+itemID.String()); err != nil {
		slog.Error("error al limpiar el cache de sabores del item", "error", err)
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_additions_cache+"item:"+itemID.String()); err != nil {
		slog.Error("error al limpiar el cache de adiciones del item", "error", err)
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_additions_cache+"catalog:shop:"+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache del catálogo de adiciones", "error", err)
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_best_sellers_cache+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache de best sellers", "error", err)
	}
}

func (p *productsRepo) clearAdditionCaches(ctx context.Context, shopID uuid.UUID, itemIDs []uuid.UUID) {
	if err := p.cache.ClearByPrefix(ctx, prefix_additions_cache+"catalog:shop:"+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache del catálogo de adiciones", "error", err)
	}

	for _, itemID := range itemIDs {
		if err := p.cache.ClearByPrefix(ctx, prefix_additions_cache+"item:"+itemID.String()); err != nil {
			slog.Error("error al limpiar el cache de adiciones del item", "error", err, "itemID", itemID)
		}
	}

	p.clearShopCache(ctx, shopID)
}

func (p *productsRepo) clearItemDiscountCaches(ctx context.Context, shopID uuid.UUID) {
	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}
	if err := p.cache.ClearByPrefix(ctx, prefix_best_sellers_cache+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}
}
