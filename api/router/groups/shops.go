package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shops/handler"
	"github.com/labstack/echo/v4"
)

type ShopsGroup interface {
	Resource(g *echo.Echo)
}

type shopGroup struct {
	handlerShops handler.Shops
	middleAuth   auth.AuthMiddleware
}

func NewShopsGroup(handlerShops handler.Shops, middleAuth auth.AuthMiddleware) ShopsGroup {
	return &shopGroup{handlerShops, middleAuth}
}

func (s *shopGroup) Resource(g *echo.Echo) {
	customerRoutes := g.Group("/shops",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.Customer),
	)
	customerRoutes.GET("/all", s.handlerShops.GetAllShops)
	customerRoutes.GET("", s.handlerShops.GetShopsBy)

	businessRoutes := g.Group("/business/shops",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.OrderManager, models.Owner),
	)
	businessRoutes.GET("/me", s.handlerShops.GetMyShop)
}
