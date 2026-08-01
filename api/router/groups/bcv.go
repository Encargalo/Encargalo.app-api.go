package groups

import (
	"Encargalo.app-api.go/internal/bcv/handler"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
)

type BcvGroup interface {
	Resource(g *echo.Echo)
}

type bcvGroup struct {
	apiKeyAuth sharedmw.ApiKeyAuth
	handler    handler.BcvHandler
}

func NewBcvGroup(apiKeyAuth sharedmw.ApiKeyAuth, handler handler.BcvHandler) BcvGroup {
	return &bcvGroup{apiKeyAuth, handler}
}

func (b *bcvGroup) Resource(g *echo.Echo) {
	group := g.Group("/bcv")

	group.GET("/rate", b.handler.GetRate, b.apiKeyAuth.Validate)
}
