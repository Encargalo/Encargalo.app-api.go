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
	UpdateItem(ctx context.Context, item dtos.UpdateItem, itemID, shopID uuid.UUID) error
	RegisterCategory(ctx context.Context, category dtos.RegisterCategory) error
	UpdateCategory(ctx context.Context, category dtos.UpdateCategory, categoryID, shopID uuid.UUID) error
	SearchProductsByShopID(ctx context.Context, shopID uuid.UUID, forCustomer bool) (dtos.CategoriesResponse, error)
	SearchProductsByShopIDForOwner(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error)
	SearchAdditionsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.AdditionsResponse, error)
	SearchAdditionsCatalogByShopID(ctx context.Context, shopID uuid.UUID) (dtos.AdditionsResponse, error)
	SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchFlavorsByCategoryID(ctx context.Context, categoryID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error)
	AddSoldItem(ctx context.Context, item models.DataProduceOrder) error
	GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error)
	RegisterFlavor(ctx context.Context, flavor dtos.RegisterFlavor) error
	UpdateFlavor(ctx context.Context, flavor dtos.UpdateFlavor, flavorID, shopID uuid.UUID) error
	DeleteFlavor(ctx context.Context, flavorID, shopID uuid.UUID) error
	SearchMacroCategories(ctx context.Context, shopType string) (dtos.MacroCategoriesResponse, error)
	// SeedDefaultCategories crea, para un shop recién creado, una categoría
	// vacía por cada macro-categoría de su tipo (Restaurant/Supermarket/Turbo).
	SeedDefaultCategories(ctx context.Context, shopID uuid.UUID, shopType string) error
	RegisterItemDiscount(ctx context.Context, discount dtos.RegisterItemDiscount) error
	UpdateItemDiscount(ctx context.Context, discount dtos.UpdateItemDiscount, discountID, shopID uuid.UUID) error
	DeleteItemDiscount(ctx context.Context, discountID, shopID uuid.UUID) error
	SearchCategorySummariesByShopID(ctx context.Context, shopID uuid.UUID, filter string) (dtos.CategorySummariesResponse, error)
	SearchCategoriesByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error)
}

type ProductsRepo interface {
	RegisterItem(context.Context, *models.Item) error
	UpdateItem(ctx context.Context, item *models.UpdateItem, shopID uuid.UUID) error
	SearchItemByID(context.Context, uuid.UUID) (*models.Item, error)
	RegisterCategory(ctx context.Context, category *models.Category) error
	SearchCategoryByID(ctx context.Context, categoryID uuid.UUID) (*models.Category, error)
	UpdateCategory(ctx context.Context, category *models.UpdateCategory, shopID uuid.UUID) error
	ReplaceCategorySchedules(ctx context.Context, categoryID, shopID uuid.UUID, schedules []models.Schedule) error
	SearchProductsByShopID(ctx context.Context, shopID uuid.UUID, forCustomer bool) (dtos.CategoriesResponse, error)
	SearchAdditionsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.AdditionsResponse, error)
	SearchAdditionsCatalogByShopID(ctx context.Context, shopID uuid.UUID) (dtos.AdditionsResponse, error)
	SearchAdditionItemsByIDs(ctx context.Context, shopID uuid.UUID, additionIDs []uuid.UUID) (models.Items, error)
	SyncAdditionSnapshots(ctx context.Context, shopID uuid.UUID, entry models.AdditionEntry) error
	SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchFlavorsByCategoryID(ctx context.Context, categoryID uuid.UUID) (dtos.FlavorsResponse, error)
	SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error)
	AddSoldItem(ctx context.Context, item models.DataProduceOrder) error
	GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error)
	RegisterFlavor(ctx context.Context, flavor models.CatalogFlavor) error
	SearchFlavorByID(ctx context.Context, flavorID uuid.UUID) (*models.CatalogFlavor, error)
	UpdateFlavor(ctx context.Context, flavorID uuid.UUID, entry models.FlavorEntry, shopID, categoryID uuid.UUID) error
	DeleteFlavor(ctx context.Context, flavor models.CatalogFlavor) error
	SearchMacroCategories(ctx context.Context, shopType string) (models.MacroCategories, error)
	RegisterItemDiscount(ctx context.Context, discount *models.ItemDiscount) error
	SearchItemDiscountByID(ctx context.Context, discountID uuid.UUID) (*models.ItemDiscount, error)
	UpdateItemDiscount(ctx context.Context, discount *models.UpdateItemDiscount, shopID, itemID uuid.UUID) error
	DeleteItemDiscount(ctx context.Context, discount *models.ItemDiscount) error
	SearchCategoriesByIDs(ctx context.Context, shopID uuid.UUID, categoryIDs []uuid.UUID) (models.Categories, error)
	SearchCategoriesByShopID(ctx context.Context, shopID uuid.UUID) (models.Categories, error)
	SearchCategorySummariesByShopID(ctx context.Context, shopID uuid.UUID, filter string) (models.Categories, error)
	IsAdditionItem(ctx context.Context, itemID uuid.UUID) (bool, error)
}
