package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"Encargalo.app-api.go/api/router"
	"Encargalo.app-api.go/cmd/server/providers"
	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	charmlog "github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
)

func main() {
	logger := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           charmlog.DebugLevel,
	})
	slog.SetDefault(slog.New(logger))

	container := providers.BuildContainer()

	if err := container.Invoke(func(server *echo.Echo, router *router.Router, config config.Config, bcv bcvPorts.BcvApp) {
		bcv.Start(context.Background())
		router.Init()
		server.Logger.Fatal(server.Start(fmt.Sprintf(":%d", config.Server.Port)))
	}); err != nil {
		panic(err)
	}

}
