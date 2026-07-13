package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"Encargalo.app-api.go/api/router"
	"Encargalo.app-api.go/cmd/server/providers"
	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/config"
	"github.com/labstack/echo/v4"
)

func main() {
	applog.SetupJSON(slog.LevelInfo)

	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("pánico del servidor", "error", fmt.Sprint(recovered))
			os.Exit(1)
		}
	}()

	if err := run(); err != nil {
		slog.Error("fallo al iniciar el servidor", "error", err.Error())
		os.Exit(1)
	}
}

func run() error {
	container := providers.BuildContainer()

	return container.Invoke(func(server *echo.Echo, router *router.Router, cfg config.Config, bcv bcvPorts.BcvApp) error {
		bcv.Start(context.Background())
		router.Init()

		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		slog.Info("servidor iniciando", "addr", addr)

		err := server.Start(addr)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("inicio de echo: %w", err)
	})
}
