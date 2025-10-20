package groups

import (
	"Encargalo.app-api.go/internal/shops/handler"
	"github.com/labstack/echo/v4"
)

type ShopsGroup interface {
	Resource(g *echo.Echo)
}

type shopGroup struct {
	handlerShops handler.Shops
}

func NewShopsGroup(handlerShops handler.Shops) ShopsGroup {
	return &shopGroup{handlerShops}
}

func (s *shopGroup) Resource(g *echo.Echo) {

	group := g.Group("/shops")

	group.GET("/all", s.handlerShops.GetAllShops)
	group.GET("", s.handlerShops.GetShopsBy)
}
