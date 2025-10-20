package main

import (
	"fmt"

	"Encargalo.app-api.go/api/router"
	"Encargalo.app-api.go/cmd/server/providers"
	"Encargalo.app-api.go/internal/shared/config"
	"github.com/labstack/echo/v4"
)

func main() {

	container := providers.BuildContainer()

	if err := container.Invoke(func(server *echo.Echo, router *router.Router, config config.Config) {
		router.Init()
		server.Logger.Fatal(server.Start(fmt.Sprintf(":%d", config.Server.Port)))
	}); err != nil {
		panic(err)
	}

}
