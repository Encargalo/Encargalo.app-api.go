package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/history/handler"
	"github.com/labstack/echo/v4"
)

type HistoryGroup interface {
	Resource(g *echo.Echo)
}

type historyGroup struct {
	middle         auth.AuthMiddleware
	handlerHistory handler.HistoryHandler
}

func NewHistoryGroup(middle auth.AuthMiddleware, handlerHistory handler.HistoryHandler) HistoryGroup {
	return &historyGroup{middle, handlerHistory}
}

func (h *historyGroup) Resource(g *echo.Echo) {
	group := g.Group("/history")

	group.GET("", h.handlerHistory.GetHistory, h.middle.Auth, h.middle.Require(models.Customer))
}
