package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/trips/handler"
	"github.com/labstack/echo/v4"
)

type TripsGroup interface {
	Resource(g *echo.Echo)
}

type tripsGroup struct {
	middle       auth.AuthMiddleware
	handlerTrips handler.TripsHandler
}

func NewTripsGroup(middle auth.AuthMiddleware, handlerTrips handler.TripsHandler) TripsGroup {
	return &tripsGroup{middle, handlerTrips}
}

func (t *tripsGroup) Resource(g *echo.Echo) {
	group := g.Group("/trips")

	group.POST("", t.handlerTrips.RequestTrip, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/history", t.handlerTrips.GetTripHistory, t.middle.Auth, t.middle.Require(models.Customer))
	group.GET("/:id", t.handlerTrips.GetTrip, t.middle.Auth, t.middle.Require(models.Customer))
	group.POST("/:id/cancel", t.handlerTrips.CancelTrip, t.middle.Auth, t.middle.Require(models.Customer))
}
