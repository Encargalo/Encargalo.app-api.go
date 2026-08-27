package groups

import (
	"net/http"
	"net/http/pprof"

	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
)

type DiagnosticsGroup interface {
	Resource(g *echo.Echo)
}

type diagnosticsGroup struct {
	gatewaySecret sharedmw.GatewaySecretAuth
}

func NewDiagnosticsGroup(gatewaySecret sharedmw.GatewaySecretAuth) DiagnosticsGroup {
	return &diagnosticsGroup{gatewaySecret}
}

// Resource expone net/http/pprof (perfiles de CPU, heap, goroutines, etc. del
// proceso) detrás del mismo secreto compartido que protege
// /internal/verify-session — no es una ruta de negocio, así que no lleva
// middleAuth.Auth/Require. Se monta exactamente en "/debug/pprof" porque
// pprof.Index hace su propio despacho por nombre de perfil (heap, goroutine,
// allocs, ...) recortando ese prefijo del path a mano; montarlo en otro
// prefijo rompería ese despacho.
func (d *diagnosticsGroup) Resource(g *echo.Echo) {
	group := g.Group("/debug/pprof", d.gatewaySecret.Validate)

	group.GET("", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	group.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	group.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	group.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	group.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.POST("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	group.GET("/:name", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
}
