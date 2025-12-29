package app

import (
	"context"

	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	productsDto "Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"

	"github.com/google/uuid"
)

type productsApp struct {
	repo        ports.ProductsRepo
	uploadimage uploadimage.UploadImage
}

func NewProductsApp(repo ports.ProductsRepo, uploadimage uploadimage.UploadImage) ports.ProductsApp {
	return &productsApp{repo, uploadimage}
}

func (p *productsApp) RegisterItem(ctx context.Context, item dtos.RegisterItem) error {

	url, err := p.uploadimage.UploadImage(ctx, item.Image)
	if err != nil {
		return err
	}

	item.Image = url

	itemModel := models.Item{}
	itemModel.BuildToModel(item)

	if err := p.repo.RegisterItem(ctx, &itemModel); err != nil {
		return err
	}

	return nil
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
