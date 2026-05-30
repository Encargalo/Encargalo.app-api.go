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
	businessRoutes.PUT("/:item_id", r.handlerProducts.UpdateItem)
}
