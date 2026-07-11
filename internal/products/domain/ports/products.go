package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"

	"github.com/google/uuid"
)

// ShopLaunchPromoProvider es implementado por shops/app para exponer la
// ventana de descuento día 1 sin que products dependa de shops/domain/ports.
type ShopLaunchPromoProvider interface {
	GetLaunchPromoEndsAt(ctx context.Context, shopID uuid.UUID) (*time.Time, error)
}

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
	RegisterFlavor(ctx context.Context, flavor dtos.RegisterFlavor) error
	UpdateFlavor(ctx context.Context, flavor dtos.UpdateFlavor, flavorID, userID uuid.UUID) error
	DeleteFlavor(ctx context.Context, flavorID, userID uuid.UUID) error
	SearchMacroCategories(ctx context.Context) (dtos.MacroCategoriesResponse, error)
	RegisterItemDiscount(ctx context.Context, discount dtos.RegisterItemDiscount) error
	UpdateItemDiscount(ctx context.Context, discount dtos.UpdateItemDiscount, discountID, userID uuid.UUID) error
	DeleteItemDiscount(ctx context.Context, discountID, userID uuid.UUID) error
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
	RegisterFlavor(ctx context.Context, flavor *models.Flavor) error
	SearchFlavorByID(ctx context.Context, flavorID uuid.UUID) (*models.Flavor, error)
	SearchFlavorsByIDs(ctx context.Context, flavorIDs []uuid.UUID) (models.Flavors, error)
	UpdateFlavor(ctx context.Context, flavor *models.UpdateFlavor, shopID, categoryID uuid.UUID) error
	DeleteFlavor(ctx context.Context, flavor *models.Flavor) error
	SearchMacroCategories(ctx context.Context) (models.MacroCategories, error)
	RegisterItemDiscount(ctx context.Context, discount *models.ItemDiscount) error
	SearchItemDiscountByID(ctx context.Context, discountID uuid.UUID) (*models.ItemDiscount, error)
	UpdateItemDiscount(ctx context.Context, discount *models.UpdateItemDiscount, shopID, itemID uuid.UUID) error
	DeleteItemDiscount(ctx context.Context, discount *models.ItemDiscount) error
}
