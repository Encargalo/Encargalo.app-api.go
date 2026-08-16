package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/handler"
	ridersHandler "Encargalo.app-api.go/internal/riders/handler"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
)

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
	handlerStaffReviews  handler.StaffReviews
	handlerRiders        ridersHandler.RiderHandler
	handlerFinance       ridersHandler.FinanceHandler
	handlerStaffRiders   ridersHandler.StaffRiderHandler
	rateLimiter          sharedmw.RateLimiter
}

func NewOrdersGroup(
	middle auth.AuthMiddleware,
	handlerOrders handler.OrderHandler,
	hanlderManagerOrders handler.ManagerOrder,
	handlerStaffReviews handler.StaffReviews,
	handlerRiders ridersHandler.RiderHandler,
	handlerFinance ridersHandler.FinanceHandler,
	handlerStaffRiders ridersHandler.StaffRiderHandler,
	rateLimiter sharedmw.RateLimiter,
) OrdersGroup {
	return &ordersGroup{middle, handlerOrders, hanlderManagerOrders, handlerStaffReviews, handlerRiders, handlerFinance, handlerStaffRiders, rateLimiter}
}

func (o *ordersGroup) Resource(g *echo.Echo) {
	group := g.Group("/orders")

	confirmRateLimit := func(route string) echo.MiddlewareFunc {
		return o.rateLimiter.LimitByIP("orders:"+route, confirmCodeRateLimitCount, confirmCodeRateLimitWindow)
	}

	group.POST("", o.handlerOrders.CreateOrder, o.middle.Auth, o.middle.Require(models.Customer))
	group.POST("/quote", o.handlerOrders.CreateQuote, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/history", o.handlerOrders.GetOrderHistory, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/manager", o.hanlderManagerOrders.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Customer, models.OrderManager, models.Owner))
	group.GET("/staff/reviews", o.handlerStaffReviews.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Staff))
	group.GET("/rider", o.handlerRiders.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Rider))
	group.POST("/:id/confirm-pickup", o.handlerOrders.ConfirmPickup, o.middle.Auth, o.middle.Require(models.OrderManager, models.Owner), confirmRateLimit("confirm-pickup"))
	group.POST("/:id/confirm-delivery", o.handlerOrders.ConfirmDelivery, o.middle.Auth, o.middle.Require(models.Rider), confirmRateLimit("confirm-delivery"))

	riders := g.Group("/riders", o.middle.Auth, o.middle.Require(models.Rider))
	riders.GET("/balance", o.handlerFinance.GetBalance)
	riders.POST("/withdrawal", o.handlerFinance.RequestWithdrawal)
	riders.GET("/transactions", o.handlerFinance.ListTransactions)

	staffRiders := g.Group("/riders/staff", o.middle.Auth, o.middle.Require(models.Staff))
	staffRiders.GET("/candidates", o.handlerStaffRiders.FindCandidate)
	staffRiders.GET("/profiles", o.handlerStaffRiders.ListProfiles)
	staffRiders.GET("/profiles/:userId", o.handlerStaffRiders.GetProfile)
	staffRiders.PUT("/profiles/:userId", o.handlerStaffRiders.UpsertProfile)
	staffRiders.DELETE("/profiles/:userId/photos/:field", o.handlerStaffRiders.DeletePhoto)
	staffRiders.POST("/profiles/:userId/activate", o.handlerStaffRiders.Activate)
	staffRiders.POST("/profiles/:userId/suspend", o.handlerStaffRiders.Suspend)
	staffRiders.POST("/profiles/:userId/touch-contact", o.handlerStaffRiders.TouchContact)
}
