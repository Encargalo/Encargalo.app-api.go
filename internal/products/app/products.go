package app

import (
	"context"

	productsDto "Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"

	"github.com/google/uuid"
)

type productsApp struct {
	repo ports.ProductsRepo
}

func NewProductsApp(repo ports.ProductsRepo) ports.ProductsApp {
	return &productsApp{repo}
}

func (p *productsApp) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (productsDto.CategoriesResponse, error) {
	return p.repo.SearchProductsByShopID(ctx, shopID)
}

func (p *productsApp) SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (productsDto.AdditionsResponse, error) {
	return p.repo.SearchAdditionsByShopID(ctx, categoryID)
}

func (p *productsApp) SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (productsDto.FlavorsResponse, error) {
	return p.repo.SearchFlavorsByItemID(ctx, itemID)
}
func (p *productsApp) SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (productsDto.ItemsResponse, error) {
	return p.repo.SearchBestSellersByShopID(ctx, shopID)
}

func (p *productsApp) AddSoldItem(ctx context.Context, item models.DataProduceOrder) error {
	return p.repo.AddSoldItem(ctx, item)
}
