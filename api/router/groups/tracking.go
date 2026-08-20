package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"Encargalo.app-api.go/internal/tracking/handler"
	"github.com/labstack/echo/v4"
)

const (
	trackingRateLimitCount  = 60
	trackingRateLimitWindow = time.Minute
)

type TrackingGroup interface {
	Resource(g *echo.Echo)
}

type trackingGroup struct {
	handler     handler.TrackingHandler
	middleAuth  auth.AuthMiddleware
	rateLimiter sharedmw.RateLimiter
}

func NewTrackingGroup(handler handler.TrackingHandler, middleAuth auth.AuthMiddleware, rateLimiter sharedmw.RateLimiter) TrackingGroup {
	return &trackingGroup{handler, middleAuth, rateLimiter}
}

func (t *trackingGroup) Resource(g *echo.Echo) {
	group := g.Group("/tracking",
		t.middleAuth.Auth,
		t.rateLimiter.LimitByKey(sessionUserIDKey, "tracking", trackingRateLimitCount, trackingRateLimitWindow),
	)

	group.POST("", t.handler.Track)
}
