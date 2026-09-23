package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/handler"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// createOrderBodyLimit da margen sobre el comprobante en base64 (~6.99MB) más el resto del JSON.
const createOrderBodyLimit = "8M"

// confirmCodeRateLimit acota los intentos de confirm-pickup/confirm-delivery
// por IP para mitigar fuerza bruta sobre el código de 4 dígitos.
const (
	confirmCodeRateLimitCount  = 10
	confirmCodeRateLimitWindow = time.Minute
)

type OrdersGroup interface {
	Resource(g *echo.Echo)
}

type ordersGroup struct {
	middle               auth.AuthMiddleware
	handlerOrders        handler.OrderHandler
	hanlderManagerOrders handler.ManagerOrder
	rateLimiter          sharedmw.RateLimiter
}

func NewOrdersGroup(
	middle auth.AuthMiddleware,
	handlerOrders handler.OrderHandler,
	hanlderManagerOrders handler.ManagerOrder,
	rateLimiter sharedmw.RateLimiter,
) OrdersGroup {
	return &ordersGroup{middle, handlerOrders, hanlderManagerOrders, rateLimiter}
}

func (o *ordersGroup) Resource(g *echo.Echo) {
	group := g.Group("/orders")

	confirmRateLimit := func(route string) echo.MiddlewareFunc {
		return o.rateLimiter.LimitByIP("orders:"+route, confirmCodeRateLimitCount, confirmCodeRateLimitWindow)
	}

	group.POST("", o.handlerOrders.CreateOrder, middleware.BodyLimit(createOrderBodyLimit), o.middle.Auth, o.middle.Require(models.Customer))
	group.POST("/quote", o.handlerOrders.CreateQuote, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/history", o.handlerOrders.GetOrderHistory, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/manager", o.hanlderManagerOrders.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Customer, models.OrderManager, models.Owner))
	group.POST("/:id/confirm-pickup", o.handlerOrders.ConfirmPickup, o.middle.Auth, o.middle.Require(models.OrderManager, models.Owner), confirmRateLimit("confirm-pickup"))
	group.POST("/:id/confirm-delivery", o.handlerOrders.ConfirmDelivery, o.middle.Auth, o.middle.Require(models.Rider), confirmRateLimit("confirm-delivery"))
}
