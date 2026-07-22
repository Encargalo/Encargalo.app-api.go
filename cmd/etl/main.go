package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"Encargalo.app-api.go/cmd/etl/providers"
	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	"Encargalo.app-api.go/internal/health/handler"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/otelsetup"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/robfig/cron/v3"
)

func main() {
	config.ETLEnvironments()
	cfg := config.GetETL()

	ctx := context.Background()
	logHandler, shutdownTelemetry, err := otelsetup.Setup(ctx, cfg.Observability, cfg.Environment.Current, reqctx.ServiceNameETL)
	if err != nil {
		slog.Error("fallo al iniciar telemetría", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("fallo al cerrar telemetría", "error", err.Error())
		}
	}()

	applog.SetupJSON(slog.LevelInfo, logHandler, reqctx.ServiceNameETL)

	container := providers.BuildContainer()

	err = container.Invoke(func(
		server *echo.Echo,
		etlApp ports.ETLApp,
		cfg config.ETLConfig,
	) error {
		server.Use(middleware.Recover())
		server.Use(middleware.RequestID())
		server.GET("/health", handler.HealthCheck)

		schedule := cfg.ETL.ResolvedSchedule()
		if schedule == "" {
			return fmt.Errorf("etl schedule desactivado; configura ETL_SCHEDULE (default %q)", config.DefaultETLSchedule)
		}

		cronRunner := cron.New()
		if _, err := cronRunner.AddFunc(schedule, func() {
			runOnce(context.Background(), etlApp)
		}); err != nil {
			return fmt.Errorf("etl schedule %q: %w", schedule, err)
		}
		cronRunner.Start()
		defer cronRunner.Stop()

		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		slog.Info("etl service starting",
			"addr", addr,
			"schedule", schedule,
			"lookback_hours", int(cfg.ETL.DefaultLookback().Hours()),
			"initial_lookback_days", int(cfg.ETL.InitialLookback().Hours()/24),
		)
		err := server.Start(addr)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})
	if err != nil {
		slog.Error("etl failed", "error", err)
		os.Exit(1)
	}
}

func runOnce(ctx context.Context, etlApp ports.ETLApp) {
	range_, err := etlApp.ResolveDateRange(ctx, nil, nil)
	if err != nil {
		slog.Error("etl schedule resolve range", "error", err)
		return
	}
	result, err := etlApp.Run(ctx, range_.From, range_.To, models.DefaultETLEntities)
	if err != nil {
		slog.Error("etl schedule run", "error", err)
		return
	}
	slog.Info("etl schedule completed", "message", result.Message, "from", range_.From, "to", range_.To)
}
