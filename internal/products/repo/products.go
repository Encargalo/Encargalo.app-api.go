package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	productPorts "Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
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

// activeDiscountRelation filtra la relación Item.Discount al único descuento
// activo y vigente del producto (el índice único parcial garantiza que a lo
// sumo una fila cumple is_active = true AND deleted_at IS NULL por item).
func activeDiscountRelation(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Where("item_discount.is_active = ? AND item_discount.deleted_at IS NULL", true)
}

func NewProductsRepo(db *bun.DB, cache cache.Cache) productPorts.ProductsRepo {
	return &productsRepo{db, cache}
}

func (p *productsRepo) RegisterItem(ctx context.Context, item *models.Item) error {
	_, err := p.db.NewInsert().Model(item).Exec(ctx)
	if err != nil {
		slog.Error("Error al registrar producto", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+item.ShopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	return err
}

func (p *productsRepo) DeleteItemFlavors(ctx context.Context, itemID uuid.UUID) error {
	if _, err := p.db.NewDelete().
		Model((*models.ItemsFlavor)(nil)).
		Where("item_id = ?", itemID).
		Exec(ctx); err != nil {
		slog.Error("Error al eliminar sabores del producto", "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (p *productsRepo) RegisterItemFlavors(ctx context.Context, itemID uuid.UUID, flavorIDs []uuid.UUID) error {
	itemsFlavors := make([]models.ItemsFlavor, len(flavorIDs))
	for i, flavorID := range flavorIDs {
		itemsFlavors[i] = models.ItemsFlavor{
			ItemID:   itemID,
			FlavorID: flavorID,
		}
	}

	if _, err := p.db.NewInsert().Model(&itemsFlavors).Exec(ctx); err != nil {
		slog.Error("Error al registrar sabores del producto", "error", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (p *productsRepo) UpdateItem(ctx context.Context, item *models.UpdateItem, shopID uuid.UUID) error {
	_, err := p.db.NewUpdate().Model(item).WherePK().Exec(ctx)
	if err != nil {
		slog.Error("Error al actualizar", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

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

func (p *productsRepo) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {

	cacheKey := prefix_shop_cashe + shopID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.CategoriesResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de los productos", "key", cacheKey)
	}

	total, err := p.db.NewSelect().
		Model((*models.Category)(nil)).
		Where("shop_id = ?", shopID).
		Count(ctx)
	if err != nil {
		slog.Error("error al contar categorías", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if total == 0 {
		return nil, errcustom.ErrNotFound
	}

	var categories models.Categories

	if err := p.db.NewSelect().
		Model(&categories).
		Where("shop_id = ?", shopID).
		Relation("Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("is_available = ?", true).Order("price ASC")
		}).
		Relation("Items.ItemRule").
		Relation("Items.Discount", activeDiscountRelation).
		Relation("Schedules").
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := categories.ToDomainDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los productos en el cache", "error", err)
	}

	return data, nil
}

func (p *productsRepo) SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (dtos.AdditionsResponse, error) {

	cacheKey := prefix_additions_cache + categoryID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.AdditionsResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de las adiciones", "key", cacheKey)
	}

	total, err := p.db.NewSelect().
		Model((*models.CategoryAddition)(nil)).
		Where("category_id = ?", categoryID).
		Count(ctx)
	if err != nil {
		slog.Error("error al contar adiciones", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if total == 0 {
		return nil, errcustom.ErrNotFound
	}

	var categoryAdditions []models.CategoryAddition

	if err := p.db.NewSelect().
		Model(&categoryAdditions).
		Relation("Addition").
		Order("price ASC").
		Where("category_id = ?", categoryID).
		Scan(ctx); err != nil {
		fmt.Println(err.Error())
		return nil, errcustom.ErrUnexpectedError
	}

	var additions models.Additions
	for _, ca := range categoryAdditions {
		if ca.Addition != nil {
			additions = append(additions, *ca.Addition)
		}
	}

	data := additions.ToDomainDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar las adiciones en el cache", "error", err)
	}

	return data, nil
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

	total, err := p.db.NewSelect().
		Model((*models.ItemsFlavor)(nil)).
		Where("item_id = ?", itemID).
		Count(ctx)
	if err != nil {
		slog.Error("error al contar sabores por item", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if total == 0 {
		return nil, errcustom.ErrNotFound
	}

	var flavors models.Flavors

	if err := p.db.NewSelect().
		Model((*models.ItemsFlavor)(nil)).
		ColumnExpr("f.*").
		Join("JOIN products.flavors AS f ON f.id = if.flavor_id").
		Where("if.item_id = ?", itemID).
		Scan(ctx, &flavors); err != nil {
		fmt.Println(err.Error())
		return nil, errcustom.ErrUnexpectedError
	}

	data := flavors.ToDomainDTO()

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

	total, err := p.db.NewSelect().
		Model((*models.Flavor)(nil)).
		Where("category_id = ?", categoryID).
		Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		slog.Error("error al contar sabores por categoría", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if total == 0 {
		return nil, errcustom.ErrNotFound
	}

	var flavors models.Flavors

	if err := p.db.NewSelect().
		Model(&flavors).
		Where("category_id = ?", categoryID).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		slog.Error("error al obtener los sabores por categoría", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := flavors.ToDomainDTO()

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
		Where("shop_id = ?", shopID).
		Where("is_available = ?", true).
		Relation("Category.Schedules").
		Relation("Discount", activeDiscountRelation).
		Order("sold desc").
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos mas vendidos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := topSeller.ToDomainDTO()

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

	err := p.db.NewRaw(`
		SELECT
			s.id::text        AS shop_id,
			s.name            AS shop_name,
			s.tag             AS shop_tag,
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
			AND s.latitude IS NOT NULL
			AND s.longitude IS NOT NULL
			AND i.is_available = true
			AND i.sold > 0
			AND 6371 * acos(
				cos(radians(?)) * cos(radians(s.latitude)) *
				cos(radians(s.longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(s.latitude))
			) <= 1
		ORDER BY i.sold DESC
		LIMIT 30
	`, lat, lon, lat).Scan(ctx, &rows)
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

// buildTrendingResponse arma la respuesta pública a partir del bundle
// (cacheado o recién consultado), calculando EffectiveRef/HasLaunchPromo en
// vivo contra la hora actual en vez de contra el momento en que se cacheó.
func buildTrendingResponse(bundle trendingCacheBundle) dtos.TrendingResponse {
	data := make(dtos.TrendingResponse, 0, len(bundle.Rows))
	for _, r := range bundle.Rows {
		shopID, _ := uuid.Parse(r.ShopID)
		itemID, _ := uuid.Parse(r.ItemID)
		categoryID, _ := uuid.Parse(r.ItemCategoryID)
		ref := float64(r.ItemPrice)
		effectiveRef := ref
		hasLaunchPromo := r.HasActiveLaunchPromo()
		if hasLaunchPromo {
			effectiveRef = money.RoundBs(ref * 0.5)
		}

		data = append(data, dtos.TrendingEntry{
			Shop: dtos.TrendingShop{
				ID:          shopID,
				Name:        r.ShopName,
				Tag:         r.ShopTag,
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

	if config.Get().Environment.Current != "production" {
		for _, r := range rows {
			shopID, err := uuid.Parse(r.ShopID)
			if err != nil {
				continue
			}
			fee[shopID] = 2.0
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

func (p *productsRepo) RegisterFlavor(ctx context.Context, flavor *models.Flavor) error {
	_, err := p.db.NewInsert().Model(flavor).Exec(ctx)
	if err != nil {
		slog.Error("error al registrar sabor", "error", err)
		return errcustom.ErrUnexpectedError
	}

	if err := p.cache.ClearByPrefix(ctx, prefix_flavors_cat_cache+flavor.CategoryID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}

	return nil
}

func (p *productsRepo) SearchFlavorByID(ctx context.Context, flavorID uuid.UUID) (*models.Flavor, error) {
	var flavor models.Flavor

	if err := p.db.NewSelect().
		Model(&flavor).
		Where("id = ?", flavorID).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		slog.Error("error al obtener el sabor", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &flavor, nil
}

func (p *productsRepo) SearchFlavorsByIDs(ctx context.Context, flavorIDs []uuid.UUID) (models.Flavors, error) {
	var flavors models.Flavors

	if err := p.db.NewSelect().
		Model(&flavors).
		Where("id IN (?)", bun.In(flavorIDs)).
		Where("deleted_at IS NULL").
		Scan(ctx); err != nil {
		slog.Error("error al obtener sabores por ids", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return flavors, nil
}

func (p *productsRepo) UpdateFlavor(ctx context.Context, flavor *models.UpdateFlavor, shopID, categoryID uuid.UUID) error {
	_, err := p.db.NewUpdate().Model(flavor).WherePK().Exec(ctx)
	if err != nil {
		slog.Error("error al actualizar el sabor", "error", err)
		return errcustom.ErrUnexpectedError
	}

	itemIDs, err := p.itemIDsByFlavor(ctx, flavor.ID)
	if err != nil {
		return err
	}

	p.clearFlavorRelatedCaches(ctx, shopID, categoryID, itemIDs)

	return nil
}

func (p *productsRepo) DeleteFlavor(ctx context.Context, flavor *models.Flavor) error {
	itemIDs, err := p.itemIDsByFlavor(ctx, flavor.ID)
	if err != nil {
		return err
	}

	err = p.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model((*models.Flavor)(nil)).
			Set("deleted_at = ?", time.Now()).
			Where("id = ?", flavor.ID).
			Exec(ctx); err != nil {
			return err
		}

		if _, err := tx.NewDelete().
			Model((*models.ItemsFlavor)(nil)).
			Where("flavor_id = ?", flavor.ID).
			Exec(ctx); err != nil {
			return err
		}

		if len(itemIDs) == 0 {
			return nil
		}

		_, err := tx.NewUpdate().
			Model((*models.Item)(nil)).
			Set("has_flavors = EXISTS (SELECT 1 FROM products.items_flavors WHERE item_id = items.id)").
			Where("id IN (?)", bun.In(itemIDs)).
			Exec(ctx)
		return err
	})
	if err != nil {
		slog.Error("error al eliminar el sabor", "error", err)
		return errcustom.ErrUnexpectedError
	}

	p.clearFlavorRelatedCaches(ctx, flavor.ShopID, flavor.CategoryID, itemIDs)

	return nil
}

// itemIDsByFlavor devuelve los items actualmente asociados a un sabor, usado
// para saber qué cachés de sabores por item invalidar y qué items recalcular
// has_flavors al eliminar el sabor.
func (p *productsRepo) itemIDsByFlavor(ctx context.Context, flavorID uuid.UUID) ([]uuid.UUID, error) {
	var itemIDs []uuid.UUID

	if err := p.db.NewSelect().
		Model((*models.ItemsFlavor)(nil)).
		Column("item_id").
		Where("flavor_id = ?", flavorID).
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

func (p *productsRepo) SearchMacroCategories(ctx context.Context) (models.MacroCategories, error) {
	cached, _ := p.cache.Get(ctx, prefix_macro_categories)
	if cached != "" {
		var data models.MacroCategories
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
		slog.Error("error al hacer unmarshal de las macro categorías", "key", prefix_macro_categories)
	}

	var macroCategories models.MacroCategories

	if err := p.db.NewSelect().
		Model(&macroCategories).
		Order("name ASC").
		Scan(ctx); err != nil {
		slog.Error("error al obtener las macro categorías", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if err := p.cache.Set(ctx, prefix_macro_categories, macroCategories, 24*time.Hour); err != nil {
		slog.Error("error al guardar las macro categorías en el cache", "error", err)
	}

	return macroCategories, nil
}

func (p *productsRepo) RegisterItemDiscount(ctx context.Context, discount *models.ItemDiscount) error {
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

func (p *productsRepo) clearItemDiscountCaches(ctx context.Context, shopID uuid.UUID) {
	if err := p.cache.ClearByPrefix(ctx, prefix_shop_cashe+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}
	if err := p.cache.ClearByPrefix(ctx, prefix_best_sellers_cache+shopID.String()); err != nil {
		slog.Error("error al limpiar el cache", "error", err)
	}
}
