package router

import (
	"net/http"

	"Encargalo.app-api.go/internal/health/handler"
	"Encargalo.app-api.go/internal/shared/config"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Router struct {
	server *echo.Echo
	config config.Config
}

func New(
	server *echo.Echo,
	config config.Config,
) *Router {
	return &Router{
		server,
		config,
	}
}

func (r *Router) Init() {

	r.server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "remote_ip=${remote_ip}, method=${method}, uri=${uri}, status=${status}, latency=${latency_human}\n\n",
	}))

	r.server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:                             []string{"*"},
		AllowMethods:                             []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:                             []string{echo.HeaderContentType},
		AllowCredentials:                         true,
		UnsafeWildcardOriginWithAllowCredentials: true,
	}))

	r.server.Use(middleware.Recover())

	r.server.GET("/health", handler.HealthCheck)
	r.server.GET("/docs/*", echoSwagger.EchoWrapHandler())

}
