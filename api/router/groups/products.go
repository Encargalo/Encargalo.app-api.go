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

	group := g.Group("/products")

	group.POST("items", r.handlerProducts.RegisterItems, r.middleAuth.Auth, r.middleAuth.Require(models.OrderManager))
	group.PUT("/:item_id", r.handlerProducts.UpdateItem, r.middleAuth.Auth, r.middleAuth.Require(models.OrderManager))
	group.GET("", r.handlerProducts.SearchProductsByShopID)
	group.GET("/best-sellers", r.handlerProducts.SearchBestSellersByShopID)
	group.GET("/additions", r.handlerProducts.SearchAdditionsByShopID)
	group.GET("/flavors", r.handlerProducts.SearchFlavorsByItemID)
	group.GET("/flavors/category", r.handlerProducts.SearchFlavorsByCategoryID)
}
