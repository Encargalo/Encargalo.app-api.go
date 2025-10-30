package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	prefix_shop_cashe         = "prod_"
	prefix_best_sellers_cache = "best_sellers_"
)

type productsRepo struct {
	db    *bun.DB
	cache cache.Cache
}

func NewProductsRepo(db *bun.DB, cache cache.Cache) ports.ProductsRepo {
	return &productsRepo{db, cache}
}

func (p *productsRepo) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {

	var prods dtos.CategoriesResponse

	products, _ := p.cache.Get(ctx, prefix_shop_cashe+shopID.String())
	if products != "" {
		err := json.Unmarshal([]byte(products), &prods)
		if err != nil {
			slog.Error("error al hacer unmarshal de los products", "error", err)
		}

		return prods, nil
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

	if len(categories) == 0 {
		return nil, errcustom.ErrNotFound
	}

	if err := p.cache.Set(ctx, prefix_shop_cashe+shopID.String(), categories.ToDomainDTO(), 1*1*time.Hour); err != nil {
		slog.Error("error al guardar los productos en el cache", "error", err)
	}

	return categories.ToDomainDTO(), nil
}

func (p *productsRepo) SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (dtos.AdditionsResponse, error) {

	var categoryAdditions []models.CategoryAddition

	err := p.db.NewSelect().
		Model(&categoryAdditions).
		Relation("Addition").
		Where("category_id = ?", categoryID).
		Scan(ctx)

	if err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("unexpected error")
	}

	if len(categoryAdditions) == 0 {
		return nil, fmt.Errorf("additions not found")
	}

	var additions models.Additions
	for _, ca := range categoryAdditions {
		if ca.Addition != nil {
			additions = append(additions, *ca.Addition)
		}
	}

	return additions.ToDomainDTO(), nil
}

func (p *productsRepo) SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error) {

	var flavors models.Flavors

	if err := p.db.NewSelect().
		Model((*models.ItemsFlavor)(nil)).
		ColumnExpr("f.*").
		Join("JOIN products.flavors AS f ON f.id = if.flavor_id").
		Where("if.item_id = ?", itemID).
		Scan(ctx, &flavors); err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("unexpected error")
	}

	if len(flavors) == 0 {
		return nil, fmt.Errorf("flavors not found")
	}

	return flavors.ToDomainDTO(), nil
}

func (p *productsRepo) SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error) {

	var items dtos.ItemsResponse

	prod, err := p.cache.Get(ctx, prefix_best_sellers_cache+shopID.String())
	if err != nil {
		slog.Error("error al obtener los productos mas vendidos del cache", "error", err)
	}

	if prod != "" {
		err := json.Unmarshal([]byte(prod), &items)
		if err != nil {
			slog.Error("error al hacer unmarshal de los productos mas vendidos", "error", err)
		}

		return items, nil
	}

	var topSeller models.Items

	if err := p.db.NewSelect().
		Model(&topSeller).
		Where("shop_id = ?", shopID).
		Order("sold desc").Limit(5).
		Scan(ctx); err != nil {
		slog.Error("error al obtener los productos mas vendidos", "error", err)
		return topSeller.ToDomainDTO(), errcustom.ErrUnexpectedError
	}

	if len(topSeller) == 0 {
		return topSeller.ToDomainDTO(), errcustom.ErrNotFound
	}

	if err := p.cache.Set(ctx, prefix_best_sellers_cache+shopID.String(), topSeller.ToDomainDTO(), 1*1*time.Hour); err != nil {
		slog.Error("error al guardar los productos mas vendidos en el cache", "error", err)
	}

	return topSeller.ToDomainDTO(), nil
}

func (p *productsRepo) AddSoldItem(ctx context.Context, item models.DataProduceOrder) error {

	_, err := p.db.NewUpdate().
		Model(&item).
		Set("sold = sold + ?", item.Quantity).
		Where("id = ?", item.ProductID).
		Exec(ctx)
	return err

}
