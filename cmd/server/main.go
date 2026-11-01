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
	appPayments "Encargalo.app-api.go/internal/payments/app"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	appRiders "Encargalo.app-api.go/internal/riders/app"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/redis"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/otelsetup"
	"Encargalo.app-api.go/internal/shared/reqctx"
	appShipments "Encargalo.app-api.go/internal/shipments/app"
	shipmentsModels "Encargalo.app-api.go/internal/shipments/domain/models"
	appShops "Encargalo.app-api.go/internal/shops/app"
	appTrips "Encargalo.app-api.go/internal/trips/app"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
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
		hub *appPayments.StaffReviewHubImpl,
		reviews paymentsPorts.PaymentReviewsApp,
		registry *appPayments.SubjectRegistry,
		orderSubject paymentsPorts.PaymentSubject,
		tripSubject *appTrips.TripPaymentSubject,
		shipmentSubject *appShipments.ShipmentPaymentSubject,
		tripSnapshot appTrips.TripStaffSnapshot,
		shipmentSnapshot appShipments.ShipmentStaffSnapshot,
		staffSnapshot appOrders.OrderStaffSnapshot,
		pendingWorker *appOrders.PendingAcceptWorker,
		paymentProofAlertWorker *appOrders.PaymentProofAlertWorker,
		tripPaymentProofAlertWorker *appTrips.TripPaymentProofAlertWorker,
		batchCloseWorker *appShops.BatchCloseWorker,
		refundHub *appCustomer.StaffRefundHubImpl,
		finance usersPorts.CustomerFinanceApp,
		refundAlertWorker *appCustomer.RefundAlertWorker,
		accountDeletionWorker *appCustomer.AccountDeletionWorker,
		orderMgr ordersPorts.AppManagerOrder,
		ridersMgr ridersPorts.RidersApp,
		streamTrimWorker *redis.StreamTrimWorker,
		tripOfferTimeoutWorker *appRiders.TripOfferTimeoutWorker,
		shipmentOfferTimeoutWorker *appRiders.ShipmentOfferTimeoutWorker,
	) error {
		registry.Register(paymentsModels.ReferenceTypeOrder, orderSubject)
		registry.Register(tripsModels.ReferenceTypeTrip, tripSubject)
		registry.Register(shipmentsModels.ReferenceTypeShipment, shipmentSubject)
		reviews.SetSnapshotProvider(staffSnapshot)
		hub.SetSnapshotProvider(reviews.ListActiveForStaff)
		hub.SetSnapshotProvider(tripSnapshot)
		hub.SetSnapshotProvider(shipmentSnapshot)
		refundHub.SetFinanceApp(finance)
		finance.SetCustomerNotifier(orderMgr)
		if setter, ok := ridersMgr.(ridersPorts.OrderCustomerNotifierSetter); ok {
			setter.SetOrderCustomerNotifier(orderMgr)
		}

		bcv.Start(context.Background())
		pendingWorker.Start(context.Background())
		paymentProofAlertWorker.Start(context.Background())
		tripPaymentProofAlertWorker.Start(context.Background())
		batchCloseWorker.Start(context.Background())
		refundAlertWorker.Start(context.Background())
		accountDeletionWorker.Start(context.Background())
		streamTrimWorker.Start(context.Background())
		tripOfferTimeoutWorker.Start(context.Background())
		shipmentOfferTimeoutWorker.Start(context.Background())
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
