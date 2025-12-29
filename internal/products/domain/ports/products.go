package ports

import (
	"context"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	productsDto "Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"

	"github.com/google/uuid"
)

type ProductsApp interface {
	RegisterItem(ctx context.Context, item dtos.RegisterItem) error
	SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (productsDto.CategoriesResponse, error)
	SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (productsDto.AdditionsResponse, error)
	SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (productsDto.FlavorsResponse, error)
	SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (productsDto.ItemsResponse, error)
	AddSoldItem(ctx context.Context, item models.DataProduceOrder) error
}

type ProductsRepo interface {
	RegisterItem(context.Context, *models.Item) error
	SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (productsDto.CategoriesResponse, error)
	SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (productsDto.AdditionsResponse, error)
	SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (productsDto.FlavorsResponse, error)
	SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (productsDto.ItemsResponse, error)
	AddSoldItem(ctx context.Context, item models.DataProduceOrder) error
}
