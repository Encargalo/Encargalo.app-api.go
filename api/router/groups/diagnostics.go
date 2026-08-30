package groups

import (
	"net/http"
	"net/http/pprof"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"github.com/labstack/echo/v4"
)

type DiagnosticsGroup interface {
	Resource(g *echo.Echo)
}

type diagnosticsGroup struct {
	middleAuth auth.AuthMiddleware
}

func NewDiagnosticsGroup(middleAuth auth.AuthMiddleware) DiagnosticsGroup {
	return &diagnosticsGroup{middleAuth}
}

func (d *diagnosticsGroup) Resource(g *echo.Echo) {
	// Traefik inyecta el gateway secret en toda petición pública, así que por sí solo no autentica a nadie; requiere sesión+Staff igual que una ruta de negocio.
	group := g.Group("/debug/pprof", d.middleAuth.Auth, d.middleAuth.Require(models.Staff))

	group.GET("", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	group.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	group.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	group.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	group.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.POST("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	group.GET("/:name", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
}
