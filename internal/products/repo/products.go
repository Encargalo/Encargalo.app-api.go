package repo

import (
	"context"
	"fmt"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type productsRepo struct {
	db *bun.DB
}

func NewProductsRepo(db *bun.DB) ports.ProductsRepo {
	return &productsRepo{db}
}

func (p *productsRepo) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {

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
