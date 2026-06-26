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
	handlerShops   handler.Shops
	handlerFinance handler.ShopFinanceHandler
	handlerStats   handler.ShopStatsHandler
	middleAuth     auth.AuthMiddleware
}

func NewShopsGroup(handlerShops handler.Shops, handlerFinance handler.ShopFinanceHandler, handlerStats handler.ShopStatsHandler, middleAuth auth.AuthMiddleware) ShopsGroup {
	return &shopGroup{handlerShops, handlerFinance, handlerStats, middleAuth}
}

func (s *shopGroup) Resource(g *echo.Echo) {
	customerRoutes := g.Group("/shops",
		s.middleAuth.Auth,
		s.middleAuth.Require(models.Customer),
	)
	customerRoutes.GET("/all", s.handlerShops.GetAllShops)
	customerRoutes.GET("/search", s.handlerShops.SearchShops)
	customerRoutes.GET("", s.handlerShops.GetShopsBy)

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
	statsRoutes.GET("/transactions/export", s.handlerStats.ExportTransactions)
}
