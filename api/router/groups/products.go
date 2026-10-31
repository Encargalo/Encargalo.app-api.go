package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/products/handler"
	"github.com/labstack/echo/v4"
)

type ProductsGroup interface {
	Resource(g *echo.Echo)
}

type productsGroup struct {
	handlerProducts      handler.ProductsHandler
	handlerStaffProducts handler.StaffProductsHandler
	middleAuth           auth.AuthMiddleware
}

func NewProductsGroup(handlerProducts handler.ProductsHandler, handlerStaffProducts handler.StaffProductsHandler, middleAuth auth.AuthMiddleware) ProductsGroup {
	return &productsGroup{handlerProducts, handlerStaffProducts, middleAuth}
}

func (r *productsGroup) Resource(g *echo.Echo) {
	customerRoutes := g.Group("/products",
		r.middleAuth.Auth,
		r.middleAuth.Require(models.Customer),
	)
	customerRoutes.GET("", r.handlerProducts.SearchProductsByShopID)
	customerRoutes.GET("/trending", r.handlerProducts.GetTrendingProducts)
	customerRoutes.GET("/home/supermarket", r.handlerProducts.GetSupermarketHome)
	customerRoutes.GET("/best-sellers", r.handlerProducts.SearchBestSellersByShopID)
	customerRoutes.GET("/additions", r.handlerProducts.SearchAdditionsByItemID)
	customerRoutes.GET("/flavors", r.handlerProducts.SearchFlavorsByItemID)
	customerRoutes.GET("/flavors/category", r.handlerProducts.SearchFlavorsByCategoryID)

	businessRoutes := g.Group("/business/products",
		r.middleAuth.Auth,
		r.middleAuth.Require(models.OrderManager, models.Owner),
	)
	businessRoutes.GET("", r.handlerProducts.GetMyProducts)
	businessRoutes.POST("/items", r.handlerProducts.RegisterItem)
	businessRoutes.POST("/categories", r.handlerProducts.RegisterCategory)
	businessRoutes.GET("/categories", r.handlerProducts.SearchCategoriesByShopID)
	businessRoutes.GET("/categories/summary", r.handlerProducts.SearchCategorySummariesByShopID)
	businessRoutes.PUT("/categories/:category_id", r.handlerProducts.UpdateCategory)
	businessRoutes.GET("/macro-categories", r.handlerProducts.SearchMacroCategories)
	businessRoutes.POST("/flavors", r.handlerProducts.RegisterFlavor)
	businessRoutes.GET("/flavors/category", r.handlerProducts.SearchFlavorsByCategoryID)
	businessRoutes.PUT("/flavors/:flavor_id", r.handlerProducts.UpdateFlavor)
	businessRoutes.DELETE("/flavors/:flavor_id", r.handlerProducts.DeleteFlavor)
	businessRoutes.POST("/discounts", r.handlerProducts.RegisterItemDiscount)
	businessRoutes.PUT("/discounts/:discount_id", r.handlerProducts.UpdateItemDiscount)
	businessRoutes.DELETE("/discounts/:discount_id", r.handlerProducts.DeleteItemDiscount)
	businessRoutes.DELETE("/items/:item_id/discount", r.handlerProducts.DeleteItemDiscountByItemID)
	businessRoutes.GET("/additions/catalog", r.handlerProducts.SearchAdditionsCatalogByShopID)
	businessRoutes.PUT("/:item_id", r.handlerProducts.UpdateItem)

	staffRoutes := g.Group("/shops/staff/:id/products",
		r.middleAuth.Auth,
		r.middleAuth.Require(models.Staff),
	)
	staffRoutes.GET("", r.handlerStaffProducts.GetShopProducts)
	staffRoutes.POST("/items", r.handlerStaffProducts.RegisterItem)
	staffRoutes.PUT("/items/:item_id", r.handlerStaffProducts.UpdateItem)
	staffRoutes.POST("/categories", r.handlerStaffProducts.RegisterCategory)
	staffRoutes.GET("/categories", r.handlerStaffProducts.SearchCategoriesByShopID)
	staffRoutes.GET("/categories/summary", r.handlerStaffProducts.SearchCategorySummariesByShopID)
	staffRoutes.PUT("/categories/:category_id", r.handlerStaffProducts.UpdateCategory)
	staffRoutes.GET("/macro-categories", r.handlerStaffProducts.SearchMacroCategories)
	staffRoutes.POST("/flavors", r.handlerStaffProducts.RegisterFlavor)
	staffRoutes.PUT("/flavors/:flavor_id", r.handlerStaffProducts.UpdateFlavor)
	staffRoutes.DELETE("/flavors/:flavor_id", r.handlerStaffProducts.DeleteFlavor)
	staffRoutes.POST("/discounts", r.handlerStaffProducts.RegisterItemDiscount)
	staffRoutes.PUT("/discounts/:discount_id", r.handlerStaffProducts.UpdateItemDiscount)
	staffRoutes.DELETE("/discounts/:discount_id", r.handlerStaffProducts.DeleteItemDiscount)
	staffRoutes.GET("/additions/catalog", r.handlerStaffProducts.SearchAdditionsCatalogByShopID)
}
