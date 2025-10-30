package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

const prefix_shop_cashe = "prod_"

type productsRepo struct {
	db    *bun.DB
	cache *redis.Client
}

func NewProductsRepo(db *bun.DB, cache *redis.Client) ports.ProductsRepo {
	return &productsRepo{db, cache}
}

func (p *productsRepo) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {

	prods, _ := p.getProductsToCache(ctx, shopID)
	if prods != nil {
		return prods, nil
	}

	var prodsResponse dtos.CategoriesResponse
	var categories models.Categories

	if err := p.db.NewSelect().
		Model(&categories).
		Where("shop_id = ?", shopID).
		Relation("Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("is_available = ?", true).Order("price ASC")
		}).
		Relation("Items.ItemRule").
		Scan(ctx); err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("unexpected error")
	}

	if len(categories) == 0 {
		return nil, fmt.Errorf("products not found")
	}

	prodsResponse = categories.ToDomainDTO()

	p.setProductsToCache(ctx, shopID, prodsResponse)

	return prodsResponse, nil
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

	var topSeller models.Items

	if err := p.db.NewSelect().
		Model(&topSeller).
		Where("shop_id = ?", shopID).
		Order("sold desc").Limit(5).
		Scan(ctx); err != nil {
		fmt.Println(err.Error())
		return topSeller.ToDomainDTO(), fmt.Errorf("unexpected error")
	}

	if len(topSeller) == 0 {
		return topSeller.ToDomainDTO(), fmt.Errorf("top sellers not found")
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

func (p *productsRepo) getProductsToCache(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {

	products, err := p.cache.Get(ctx, prefix_shop_cashe+shopID.String()).Result()
	if err != nil {
		fmt.Println(err.Error())
	}

	if products != "" {
		var prods dtos.CategoriesResponse

		err := json.Unmarshal([]byte(products), &prods)
		if err != nil {
			fmt.Println(err.Error())
			return nil, fmt.Errorf("unexpected error")
		}

		return prods, nil
	}

	return nil, fmt.Errorf("products not found")
}

func (p *productsRepo) setProductsToCache(ctx context.Context, shopID uuid.UUID, prods dtos.CategoriesResponse) {
	products, err := json.Marshal(prods)
	if err != nil {
		slog.Error("error al hacer marshal de los products", "error", err)
	}

	if err = p.cache.Set(ctx, prefix_shop_cashe+shopID.String(), products, 1*1*time.Hour).
		Err(); err != nil {
		slog.Error("error al guardar el chache de los productos", "error", err)
	}
}
