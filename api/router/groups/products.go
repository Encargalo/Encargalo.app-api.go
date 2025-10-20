package groups

import (
	"Encargalo.app-api.go/internal/products/handler"
	"github.com/labstack/echo/v4"
)

type ProductsGroup interface {
	Resource(g *echo.Echo)
}

type productsGroup struct {
	handlerProducts handler.ProductsHandler
}

func NewProductsGroup(handlerProducts handler.ProductsHandler) ProductsGroup {
	return &productsGroup{handlerProducts}
}

func (r *productsGroup) Resource(g *echo.Echo) {

	group := g.Group("/products")

	group.GET("", r.handlerProducts.SearchProductsByShopID)
	group.GET("/best-sellers", r.handlerProducts.SearchBestSellersByShopID)
	group.GET("/additions", r.handlerProducts.SearchAdditionsByShopID)
	group.GET("/flavors", r.handlerProducts.SearchFlavorsByItemID)
}
