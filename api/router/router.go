package router

import (
	"net/http"

	"Encargalo.app-api.go/api/router/groups"
	"Encargalo.app-api.go/internal/health/handler"
	"Encargalo.app-api.go/internal/shared/config"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"Encargalo.app-api.go/internal/shared/reqctx"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "Encargalo.app-api.go/docs"

	echootel "github.com/labstack/echo-otel/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Router struct {
	server *echo.Echo
	config config.Config

	authGroup           groups.AuthGroup
	shopsGroup          groups.ShopsGroup
	productsGroup       groups.ProductsGroup
	userGroup           groups.UsersGroup
	ordersGroup         groups.OrdersGroup
	paymentsGroup       groups.PaymentsGroup
	ridersGroup         groups.RidersGroup
	paymentMethodsGroup groups.PaymentMethodsGroup
	bcvGroup            groups.BcvGroup
	mapsGroup           groups.MapsGroup
	trackingGroup       groups.TrackingGroup
	diagnosticsGroup    groups.DiagnosticsGroup
	tripsGroup          groups.TripsGroup
	bannersGroup        groups.BannersGroup
	shipmentsGroup      groups.ShipmentsGroup
	historyGroup        groups.HistoryGroup
}

func New(
	server *echo.Echo,
	config config.Config,
	authGroup groups.AuthGroup,
	shopsGroup groups.ShopsGroup,
	productsGroup groups.ProductsGroup,
	userGroup groups.UsersGroup,
	ordersGroup groups.OrdersGroup,
	paymentsGroup groups.PaymentsGroup,
	ridersGroup groups.RidersGroup,
	paymentMethodsGroup groups.PaymentMethodsGroup,
	bcvGroup groups.BcvGroup,
	mapsGroup groups.MapsGroup,
	trackingGroup groups.TrackingGroup,
	diagnosticsGroup groups.DiagnosticsGroup,
	tripsGroup groups.TripsGroup,
	bannersGroup groups.BannersGroup,
	shipmentsGroup groups.ShipmentsGroup,
	historyGroup groups.HistoryGroup,
) *Router {
	return &Router{
		server,
		config,

		authGroup,
		shopsGroup,
		productsGroup,
		userGroup,
		ordersGroup,
		paymentsGroup,
		ridersGroup,
		paymentMethodsGroup,
		bcvGroup,
		mapsGroup,
		trackingGroup,
		diagnosticsGroup,
		tripsGroup,
		bannersGroup,
		shipmentsGroup,
		historyGroup,
	}
}

func (r *Router) Init() {

	r.server.Use(sharedmw.RequestID())
	r.server.Use(echootel.NewMiddleware(reqctx.ServiceName))
	r.server.Use(sharedmw.RequestLog())
	r.server.Use(middleware.Recover())

	r.server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:                             []string{"*"},
		AllowMethods:                             []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:                             []string{echo.HeaderContentType, echo.HeaderXRequestID},
		AllowCredentials:                         true,
		UnsafeWildcardOriginWithAllowCredentials: true,
	}))

	r.server.GET("/health", handler.HealthCheck)
	r.server.GET("/docs/*", echoSwagger.EchoWrapHandler())

	r.authGroup.Resource(r.server)
	r.shopsGroup.Resource(r.server)
	r.productsGroup.Resource(r.server)
	r.userGroup.Resource(r.server)
	r.ordersGroup.Resource(r.server)
	r.paymentsGroup.Resource(r.server)
	r.ridersGroup.Resource(r.server)
	r.paymentMethodsGroup.Resource(r.server)
	r.bcvGroup.Resource(r.server)
	r.mapsGroup.Resource(r.server)
	r.trackingGroup.Resource(r.server)
	r.diagnosticsGroup.Resource(r.server)
	r.tripsGroup.Resource(r.server)
	r.bannersGroup.Resource(r.server)
	r.shipmentsGroup.Resource(r.server)
	r.historyGroup.Resource(r.server)

}
