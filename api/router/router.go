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

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

type Router struct {
	server *echo.Echo
	config config.Config

	authGroup           groups.AuthGroup
	shopsGroup          groups.ShopsGroup
	productsGroup       groups.ProductsGroup
	userGroup           groups.UsersGroup
	ordersGroup         groups.OrdersGroup
	paymentMethodsGroup groups.PaymentMethodsGroup
	bcvGroup            groups.BcvGroup
	mapsGroup           groups.MapsGroup
}

func New(
	server *echo.Echo,
	config config.Config,
	authGroup groups.AuthGroup,
	shopsGroup groups.ShopsGroup,
	productsGroup groups.ProductsGroup,
	userGroup groups.UsersGroup,
	ordersGroup groups.OrdersGroup,
	paymentMethodsGroup groups.PaymentMethodsGroup,
	bcvGroup groups.BcvGroup,
	mapsGroup groups.MapsGroup,
) *Router {
	return &Router{
		server,
		config,

		authGroup,
		shopsGroup,
		productsGroup,
		userGroup,
		ordersGroup,
		paymentMethodsGroup,
		bcvGroup,
		mapsGroup,
	}
}

func (r *Router) Init() {

	r.server.Use(sharedmw.RequestID())
	r.server.Use(otelecho.Middleware(reqctx.ServiceName))
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
	r.paymentMethodsGroup.Resource(r.server)
	r.bcvGroup.Resource(r.server)
	r.mapsGroup.Resource(r.server)

}
