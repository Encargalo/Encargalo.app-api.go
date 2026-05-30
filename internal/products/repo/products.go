package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	productPorts "Encargalo.app-api.go/internal/products/domain/ports"
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
)

type productsRepo struct {
	db       *bun.DB
	cache    cache.Cache
	bcvCache ports.BcvCache
}

func NewProductsRepo(db *bun.DB, cache cache.Cache, bcvCache ports.BcvCache) productPorts.ProductsRepo {
	return &productsRepo{db, cache, bcvCache}
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

func (p *productsRepo) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {

	cacheKey := prefix_shop_cashe + shopID.String()

	cached, _ := p.cache.Get(ctx, cacheKey)
	if cached != "" {
		var data dtos.CategoriesResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			applyRateToCategories(p.bcvRate(ctx), &data)
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
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := categories.ToDomainDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los productos en el cache", "error", err)
	}

	applyRateToCategories(p.bcvRate(ctx), &data)
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

	applyRateToAdditions(p.bcvRate(ctx), &data)
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
			applyRateToItems(p.bcvRate(ctx), &data)
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
		Order("sold desc").
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos mas vendidos", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := topSeller.ToDomainDTO()

	if err := p.cache.Set(ctx, cacheKey, data, 1*time.Hour); err != nil {
		slog.Error("error al guardar los productos mas vendidos en el cache", "error", err)
	}

	applyRateToItems(p.bcvRate(ctx), &data)
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

func (p *productsRepo) GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error) {
	lat, lon := coords.Latitude, coords.Longitude
	cacheKey := fmt.Sprintf("%s%v_%v", prefix_trending_cache, lat, lon)

	if cached, err := p.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var data dtos.TrendingResponse
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			applyRateToTrending(p.bcvRate(ctx), &data)
			return data, nil
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
			i.id::text        AS item_id,
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
		LIMIT 10
	`, lat, lon, lat).Scan(ctx, &rows)
	if err != nil {
		slog.Error("error al obtener trending products", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	data := make(dtos.TrendingResponse, 0, len(rows))
	for _, r := range rows {
		shopID, _ := uuid.Parse(r.ShopID)
		itemID, _ := uuid.Parse(r.ItemID)
		data = append(data, dtos.TrendingEntry{
			Shop: dtos.TrendingShop{
				ID:      shopID,
				Name:    r.ShopName,
				Tag:     r.ShopTag,
				Logo:    r.ShopLogo,
				Address: r.ShopAddress,
				Score:   r.ShopScore,
				Type:    r.ShopType,
			},
			Product: dtos.TrendingProduct{
				ID:          itemID,
				Name:        r.ItemName,
				Ref:         float64(r.ItemPrice),
				Image:       r.ItemImage,
				Description: r.ItemDesc,
				Score:       r.ItemScore,
				Sold:        r.ItemSold,
			},
		})
	}

	if err := p.cache.Set(ctx, cacheKey, data, 15*time.Minute); err != nil {
		slog.Error("error al guardar trending en cache", "error", err)
	}

	applyRateToTrending(p.bcvRate(ctx), &data)
	return data, nil
}

func applyRateToTrending(rate float64, data *dtos.TrendingResponse) {
	for i := range *data {
		(*data)[i].Product.Bs = (*data)[i].Product.Ref * rate
	}
}

func (p *productsRepo) bcvRate(ctx context.Context) float64 {
	rate, err := p.bcvCache.GetLatest(ctx)
	if err != nil {
		slog.Warn("bcv rate unavailable, bs prices will be 0", "error", err)
		return 0
	}
	return rate.Rate
}

func applyRateToCategories(rate float64, data *dtos.CategoriesResponse) {
	for i := range *data {
		for j := range (*data)[i].Items {
			(*data)[i].Items[j].Bs = (*data)[i].Items[j].Ref * rate
		}
	}
}

func applyRateToItems(rate float64, data *dtos.ItemsResponse) {
	for i := range *data {
		(*data)[i].Bs = (*data)[i].Ref * rate
	}
}

func applyRateToAdditions(rate float64, data *dtos.AdditionsResponse) {
	for i := range *data {
		(*data)[i].Bs = (*data)[i].Ref * rate
	}
}
