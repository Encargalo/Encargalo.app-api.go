package ports

import (
	"context"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"

	"github.com/google/uuid"
)

type ProductsApp interface {
	RegisterItem(ctx context.Context, item dtos.RegisterItem) error
	UpdateItem(ctx context.Context, item dtos.UpdateItem, itemID, userID uuid.UUID) error
	RegisterCategory(ctx context.Context, category dtos.RegisterCategory) error
	UpdateCategory(ctx context.Context, category dtos.UpdateCategory, categoryID, userID uuid.UUID) error
	SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error)
	SearchProductsByShopIDForOwner(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error)
	SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (dtos.AdditionsResponse, error)
	SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchFlavorsByCategoryID(ctx context.Context, categoryID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error)
	AddSoldItem(ctx context.Context, item models.DataProduceOrder) error
	GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error)
}

type ProductsRepo interface {
	RegisterItem(context.Context, *models.Item) error
	RegisterItemFlavors(ctx context.Context, itemID uuid.UUID, flavorIDs []uuid.UUID) error
	DeleteItemFlavors(ctx context.Context, itemID uuid.UUID) error
	UpdateItem(ctx context.Context, item *models.UpdateItem, shopID uuid.UUID) error
	SearchItemByID(context.Context, uuid.UUID) (*models.Item, error)
	RegisterCategory(ctx context.Context, category *models.Category) error
	SearchCategoryByID(ctx context.Context, categoryID uuid.UUID) (*models.Category, error)
	UpdateCategory(ctx context.Context, category *models.UpdateCategory, shopID uuid.UUID) error
	ReplaceCategorySchedules(ctx context.Context, categoryID, shopID uuid.UUID, schedules []models.Schedule) error
	SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error)
	SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (dtos.AdditionsResponse, error)
	SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchFlavorsByCategoryID(ctx context.Context, categoryID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error)
	AddSoldItem(ctx context.Context, item models.DataProduceOrder) error
	GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error)
}
