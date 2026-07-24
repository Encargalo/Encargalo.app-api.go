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
	appOrders "Encargalo.app-api.go/internal/orders/app"
	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/redis"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/otelsetup"
	"Encargalo.app-api.go/internal/shared/reqctx"
	appShops "Encargalo.app-api.go/internal/shops/app"
	appCustomer "Encargalo.app-api.go/internal/users/app"
	usersPorts "Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/labstack/echo/v4"
)

func main() {
	config.Environments()
	cfg := config.Get()

	ctx := context.Background()
	logHandler, shutdownTelemetry, err := otelsetup.Setup(ctx, cfg.Observability, cfg.Environment.Current, reqctx.ServiceName)
	if err != nil {
		slog.Error("fallo al iniciar telemetría", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("fallo al cerrar telemetría", "error", err.Error())
		}
	}()

	applog.SetupJSON(slog.LevelInfo, logHandler, reqctx.ServiceName)

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

	return container.Invoke(func(
		server *echo.Echo,
		router *router.Router,
		cfg config.Config,
		bcv bcvPorts.BcvApp,
		hub *appOrders.StaffReviewHubImpl,
		reviews ordersPorts.PaymentReviewsApp,
		pendingWorker *appOrders.PendingAcceptWorker,
		batchCloseWorker *appShops.BatchCloseWorker,
		refundHub *appCustomer.StaffRefundHubImpl,
		finance usersPorts.CustomerFinanceApp,
		refundAlertWorker *appCustomer.RefundAlertWorker,
		orderMgr ordersPorts.AppManagerOrder,
		streamTrimWorker *redis.StreamTrimWorker,
	) error {
		hub.SetReviewsApp(reviews)
		refundHub.SetFinanceApp(finance)
		finance.SetCustomerNotifier(orderMgr)

		bcv.Start(context.Background())
		pendingWorker.Start(context.Background())
		batchCloseWorker.Start(context.Background())
		refundAlertWorker.Start(context.Background())
		streamTrimWorker.Start(context.Background())
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
