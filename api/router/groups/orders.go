package groups

import (
	"Encargalo.app-api.go/internal/orders/handler"
	customerauth "Encargalo.app-api.go/internal/shared/middleware/customerAuth"
	"github.com/labstack/echo/v4"
)

type OrdersGroup interface {
	Resource(g *echo.Echo)
}

type ordersGroup struct {
	middle        customerauth.AuthMiddleware
	handlerOrders handler.OrderHandler
}

func NewOrdersGroup(middle customerauth.AuthMiddleware, handlerOrders handler.OrderHandler) OrdersGroup {
	return &ordersGroup{middle, handlerOrders}
}

func (o *ordersGroup) Resource(g *echo.Echo) {
	group := g.Group("/orders")
	group.POST("", o.handlerOrders.CreateOrder, o.middle.Auth)
}
