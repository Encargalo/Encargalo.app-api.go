package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"Encargalo.app-api.go/internal/trips/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// createTripBodyLimit da margen sobre el comprobante en base64 (~6.99MB) más el resto del JSON.
const createTripBodyLimit = "8M"

// fareEstimateIPRateLimitCount/Window y fareEstimateDailyLimitPerUser mismo
// criterio que mapsIPRateLimitCount/mapsDailyLimitPerUser (api/router/groups/maps.go):
// fare-estimate también dispara una llamada real a Google Maps en producción.
const (
	fareEstimateIPRateLimitCount  = 30
	fareEstimateIPRateLimitWindow = time.Minute

	fareEstimateDailyLimitPerUser = 200
	fareEstimateDailyLimitWindow  = 24 * time.Hour

	// nearbyRiders se consulta en polling (~cada 5 s) mientras el mapa de
	// cotización está abierto. No llama a Google, así que el tope es por minuto:
	// 20/usuario deja margen sobre 12/min, y 60/IP cubre varias personas en la misma red.
	nearbyRidersUserRateLimitCount  = 20
	nearbyRidersUserRateLimitWindow = time.Minute
	nearbyRidersIPRateLimitCount    = 60
	nearbyRidersIPRateLimitWindow   = time.Minute
)

type TripsGroup interface {
	Resource(g *echo.Echo)
}

type tripsGroup struct {
	middle               auth.AuthMiddleware
	handlerTrips         handler.TripsHandler
	handlerZoneFareRules handler.StaffZoneFareRulesHandler
	rateLimiter          sharedmw.RateLimiter
}

func NewTripsGroup(
	middle auth.AuthMiddleware,
	handlerTrips handler.TripsHandler,
	handlerZoneFareRules handler.StaffZoneFareRulesHandler,
	rateLimiter sharedmw.RateLimiter,
) TripsGroup {
	return &tripsGroup{middle, handlerTrips, handlerZoneFareRules, rateLimiter}
}

func (t *tripsGroup) Resource(g *echo.Echo) {
	group := g.Group("/trips")

	group.POST("", t.handlerTrips.RequestTrip, middleware.BodyLimit(createTripBodyLimit), t.middle.Auth, t.middle.Require(models.Customer))
	group.POST("/fare-estimate", t.handlerTrips.EstimateFare, t.middle.Auth, t.middle.Require(models.Customer),
		t.rateLimiter.LimitByIP("trips:fare-estimate", fareEstimateIPRateLimitCount, fareEstimateIPRateLimitWindow),
		t.rateLimiter.LimitByKey(sessionUserIDKey, "trips:fare-estimate:daily", fareEstimateDailyLimitPerUser, fareEstimateDailyLimitWindow),
	)
	group.GET("/services", t.handlerTrips.Services, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/history", t.handlerTrips.GetTripHistory, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/live", t.handlerTrips.Live, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/nearby-riders", t.handlerTrips.NearbyRiders, t.middle.Auth, t.middle.Require(models.Customer),
		t.rateLimiter.LimitByIP("trips:nearby-riders", nearbyRidersIPRateLimitCount, nearbyRidersIPRateLimitWindow),
		t.rateLimiter.LimitByKey(sessionUserIDKey, "trips:nearby-riders", nearbyRidersUserRateLimitCount, nearbyRidersUserRateLimitWindow),
	)
	group.GET("/:id", t.handlerTrips.GetTrip, t.middle.Auth, t.middle.Require(models.Customer))
	group.POST("/:id/cancel", t.handlerTrips.CancelTrip, t.middle.Auth, t.middle.Require(models.Customer))

	staffGroup := g.Group("/trips/staff/zone-fare-rules", t.middle.Auth, t.middle.Require(models.Staff))
	staffGroup.POST("", t.handlerZoneFareRules.CreateZoneFareRule)
	staffGroup.GET("", t.handlerZoneFareRules.ListZoneFareRules)
	staffGroup.GET("/:zone_id", t.handlerZoneFareRules.GetZoneFareRule)
	staffGroup.PATCH("/:zone_id", t.handlerZoneFareRules.UpdateZoneFareRule)
	staffGroup.DELETE("/:zone_id", t.handlerZoneFareRules.DeactivateZoneFareRule)
}
