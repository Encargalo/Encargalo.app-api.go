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
	handlerProducts handler.ProductsHandler
	middleAuth      auth.AuthMiddleware
}

func NewProductsGroup(handlerProducts handler.ProductsHandler, middleAuth auth.AuthMiddleware) ProductsGroup {
	return &productsGroup{handlerProducts, middleAuth}
}

func (r *productsGroup) Resource(g *echo.Echo) {
	customerRoutes := g.Group("/products",
		r.middleAuth.Auth,
		r.middleAuth.Require(models.Customer),
	)
	customerRoutes.GET("", r.handlerProducts.SearchProductsByShopID)
	customerRoutes.GET("/trending", r.handlerProducts.GetTrendingProducts)
	customerRoutes.GET("/best-sellers", r.handlerProducts.SearchBestSellersByShopID)
	customerRoutes.GET("/additions", r.handlerProducts.SearchAdditionsByShopID)
	customerRoutes.GET("/flavors", r.handlerProducts.SearchFlavorsByItemID)
	customerRoutes.GET("/flavors/category", r.handlerProducts.SearchFlavorsByCategoryID)

	businessRoutes := g.Group("/business/products",
		r.middleAuth.Auth,
		r.middleAuth.Require(models.OrderManager, models.Owner),
	)
	businessRoutes.GET("", r.handlerProducts.GetMyProducts)
	businessRoutes.POST("/items", r.handlerProducts.RegisterItem)
	businessRoutes.POST("/categories", r.handlerProducts.RegisterCategory)
	businessRoutes.PUT("/categories/:category_id", r.handlerProducts.UpdateCategory)
	businessRoutes.GET("/macro-categories", r.handlerProducts.SearchMacroCategories)
	businessRoutes.POST("/flavors", r.handlerProducts.RegisterFlavor)
	businessRoutes.PUT("/flavors/:flavor_id", r.handlerProducts.UpdateFlavor)
	businessRoutes.DELETE("/flavors/:flavor_id", r.handlerProducts.DeleteFlavor)
	businessRoutes.POST("/discounts", r.handlerProducts.RegisterItemDiscount)
	businessRoutes.PUT("/discounts/:discount_id", r.handlerProducts.UpdateItemDiscount)
	businessRoutes.DELETE("/discounts/:discount_id", r.handlerProducts.DeleteItemDiscount)
	businessRoutes.PUT("/:item_id", r.handlerProducts.UpdateItem)
}
