package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/maps/handler"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
)

const (
	// mapsIPRateLimitCount/Window acota, por IP, un abuso rápido (script sin
	// login válido reusando la misma IP, o un cliente en loop).
	mapsIPRateLimitCount  = 30
	mapsIPRateLimitWindow = time.Minute

	// mapsDailyLimitPerUser topa, por usuario autenticado, cuántas llamadas a
	// Google Maps puede generar en un día — el límite por IP no alcanza para
	// acotar el costo real: una sola cuenta puede sostenerlo 24/7. 500/día
	// cubre con margen un uso normal (agregar varias direcciones en el día)
	// sin dejar la cuota abierta indefinidamente.
	mapsDailyLimitPerUser = 500
	mapsDailyLimitWindow  = 24 * time.Hour
)

type MapsGroup interface {
	Resource(g *echo.Echo)
}

type mapsGroup struct {
	handler     handler.MapsHandler
	middleAuth  auth.AuthMiddleware
	rateLimiter sharedmw.RateLimiter
}

func NewMapsGroup(handler handler.MapsHandler, middleAuth auth.AuthMiddleware, rateLimiter sharedmw.RateLimiter) MapsGroup {
	return &mapsGroup{handler, middleAuth, rateLimiter}
}

func (m *mapsGroup) Resource(g *echo.Echo) {
	group := g.Group("/maps",
		m.middleAuth.Auth,
		m.middleAuth.Require(models.Customer),
		m.rateLimiter.LimitByIP("maps", mapsIPRateLimitCount, mapsIPRateLimitWindow),
		m.rateLimiter.LimitByKey(sessionUserIDKey, "maps:daily", mapsDailyLimitPerUser, mapsDailyLimitWindow),
	)

	group.GET("/geocode/reverse", m.handler.ReverseGeocode)
	group.GET("/places/autocomplete", m.handler.AutocompletePlaces)
	group.GET("/places/details", m.handler.GetPlaceDetails)
	group.GET("/places/popular", m.handler.PopularPlaces)
	group.GET("/directions", m.handler.Directions)
}

// sessionUserIDKey lee el user_id de la sesión que Auth ya cargó en el
// contexto del request — debe encadenarse después de m.middleAuth.Auth.
func sessionUserIDKey(c echo.Context) (string, bool) {
	session, ok := c.Request().Context().Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		return "", false
	}
	return session.UserID.String(), true
}
