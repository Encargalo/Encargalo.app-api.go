package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"Encargalo.app-api.go/internal/shops/handler"
	"github.com/labstack/echo/v4"
)

// candidatesRateLimit acota las búsquedas de candidato a owner por teléfono:
// el endpoint expone PII de cualquier cuenta y sería un vector de enumeración
// si se abusa desde una sesión de staff.
const (
	candidatesRateLimitCount  = 30
	candidatesRateLimitWindow = time.Minute
)

type ShopsGroup interface {
	Resource(g *echo.Echo)
}

type shopGroup struct {
	handlerShops      handler.Shops
	handlerFinance    handler.ShopFinanceHandler
	handlerStats      handler.ShopStatsHandler
	handlerStaffShops handler.StaffShopHandler
	middleAuth        auth.AuthMiddleware
	rateLimiter       sharedmw.RateLimiter
}

func NewShopsGroup(handlerShops handler.Shops, handlerFinance handler.ShopFinanceHandler, handlerStats handler.ShopStatsHandler, handlerStaffShops handler.StaffShopHandler, middleAuth auth.AuthMiddleware, rateLimiter sharedmw.RateLimiter) ShopsGroup {
	return &shopGroup{handlerShops, handlerFinance, handlerStats, handlerStaffShops, middleAuth, rateLimiter}
}

func (s *shopGroup) Resource(g *echo.Echo) {
	customerRoutes := g.Group("/shops",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.Customer),
	)
	customerRoutes.GET("/all", s.handlerShops.GetAllShops)
	customerRoutes.GET("/search", s.handlerShops.SearchShops)
	customerRoutes.GET("", s.handlerShops.GetShopsBy)

	turboRoutes := g.Group("/turbo",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.Customer),
	)
	turboRoutes.GET("/entry", s.handlerShops.GetTurboEntry)

	businessRoutes := g.Group("/business/shops",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.OrderManager, models.Owner),
	)
	businessRoutes.GET("/me", s.handlerShops.GetMyShop)
	businessRoutes.GET("/balance", s.handlerFinance.GetBalance)
	businessRoutes.GET("/transactions", s.handlerFinance.ListTransactions)

	ownerRoutes := g.Group("/business/shops",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.Owner),
	)
	ownerRoutes.PATCH("/me/status", s.handlerShops.SetOpenClose)

	statsRoutes := g.Group("/business/shops/stats",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.OrderManager, models.Owner),
	)
	statsRoutes.GET("/summary", s.handlerStats.GetSummary)
	statsRoutes.GET("/top-products", s.handlerStats.GetTopProducts)
	statsRoutes.GET("/sales-over-time", s.handlerStats.GetSalesOverTime)
	statsRoutes.GET("/customers", s.handlerStats.GetCustomerStats)
	statsRoutes.GET("/transactions/export", s.handlerStats.ExportTransactions)

	staffRoutes := g.Group("/shops/staff",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.Staff),
	)
	staffRoutes.GET("/candidates", s.handlerStaffShops.FindOwnerCandidate,
		s.rateLimiter.LimitByIP("shops:staff:candidates", candidatesRateLimitCount, candidatesRateLimitWindow))
	staffRoutes.POST("", s.handlerStaffShops.CreateDraftShop)
	staffRoutes.GET("", s.handlerStaffShops.ListShops)
	staffRoutes.GET("/:id", s.handlerStaffShops.GetShop)
	staffRoutes.PATCH("/:id", s.handlerStaffShops.UpdateDraftShop)
	staffRoutes.POST("/:id/activate", s.handlerStaffShops.ActivateShop)
}
