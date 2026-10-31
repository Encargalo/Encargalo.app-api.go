package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"Encargalo.app-api.go/internal/shipments/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// createShipmentBodyLimit da margen sobre el comprobante en base64 (~6.99MB) más el resto del JSON.
const createShipmentBodyLimit = "8M"

// Mismo criterio que trips: fare-estimate dispara una llamada real a Google Maps en producción.
const (
	shipmentFareEstimateIPRateLimitCount  = 30
	shipmentFareEstimateIPRateLimitWindow = time.Minute

	shipmentFareEstimateDailyLimitPerUser = 200
	shipmentFareEstimateDailyLimitWindow  = 24 * time.Hour
)

type ShipmentsGroup interface {
	Resource(g *echo.Echo)
}

type shipmentsGroup struct {
	middle           auth.AuthMiddleware
	handlerShipments handler.ShipmentsHandler
	rateLimiter      sharedmw.RateLimiter
}

func NewShipmentsGroup(
	middle auth.AuthMiddleware,
	handlerShipments handler.ShipmentsHandler,
	rateLimiter sharedmw.RateLimiter,
) ShipmentsGroup {
	return &shipmentsGroup{middle, handlerShipments, rateLimiter}
}

func (t *shipmentsGroup) Resource(g *echo.Echo) {
	group := g.Group("/shipments")

	group.POST("", t.handlerShipments.RequestShipment, middleware.BodyLimit(createShipmentBodyLimit), t.middle.Auth, t.middle.Require(models.Customer))
	group.POST("/fare-estimate", t.handlerShipments.EstimateFare, t.middle.Auth, t.middle.Require(models.Customer),
		t.rateLimiter.LimitByIP("shipments:fare-estimate", shipmentFareEstimateIPRateLimitCount, shipmentFareEstimateIPRateLimitWindow),
		t.rateLimiter.LimitByKey(sessionUserIDKey, "shipments:fare-estimate:daily", shipmentFareEstimateDailyLimitPerUser, shipmentFareEstimateDailyLimitWindow),
	)
	group.GET("/history", t.handlerShipments.GetShipmentHistory, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/live", t.handlerShipments.Live, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/:id", t.handlerShipments.GetShipment, t.middle.Auth, t.middle.Require(models.Customer))
	group.POST("/:id/cancel", t.handlerShipments.CancelShipment, t.middle.Auth, t.middle.Require(models.Customer))
}
