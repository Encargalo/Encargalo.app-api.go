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
	redisStream "Encargalo.app-api.go/internal/shared/adapters/redis"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/otelsetup"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
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

		// El cron ya no sincroniza CubeOrder: esa entidad se resincroniza por
		// evento (ver startOrderEventConsumer) en cuanto una orden se crea o
		// cambia de estado, sin esperar al batch horario.
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

		go startOrderEventConsumer(context.Background(), cfg, etlApp)

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
	result, err := etlApp.Run(ctx, range_.From, range_.To, models.CronETLEntities)
	if err != nil {
		slog.Error("etl schedule run", "error", err)
		return
	}
	slog.Info("etl schedule completed", "message", result.Message, "from", range_.From, "to", range_.To)
}

// orderSyncEventTypes son los tipos de mensaje del stream de orders que
// implican que CubeOrder puede haber cambiado. El stream es compartido con
// otros consumidores (cmd/notifications): cualquier otro tipo se ignora y
// se confirma igual, no es un error.
var orderSyncEventTypes = map[string]bool{
	"order_created":        true,
	"order_status_changed": true,
	"order_auto_rejected":  true,
}

// fareQuoteSyncEventTypes: las cotizaciones de viaje son inmutables, un solo
// tipo de evento basta (no hay "changed").
var fareQuoteSyncEventTypes = map[string]bool{
	"fare_quote_created": true,
}

// deliveryQuoteSyncEventTypes: igual que las de viaje, las cotizaciones de
// delivery son inmutables.
var deliveryQuoteSyncEventTypes = map[string]bool{
	"delivery_quote_created": true,
}

// tripSyncEventTypes son los tipos de mensaje del stream de trips que
// implican que CubeTrip puede haber cambiado. trip_status_changed lo publican
// tanto trips/app (RequestTrip, CancelTrip) como riders/app (accept_trip,
// advance_trip) — ver "Trips & Riders" en CLAUDE.md. trip_no_driver también
// cambia el status a no_driver (expireTripIfStillSearching).
var tripSyncEventTypes = map[string]bool{
	"trip_created":        true,
	"trip_status_changed": true,
	"trip_no_driver":      true,
}

// shopSyncEventTypes son los tipos de mensaje que implican que DimShop
// puede haber cambiado (alta, edición de datos, activación, abrir/cerrar).
var shopSyncEventTypes = map[string]bool{
	"shop_updated": true,
}

// riderSyncEventTypes son los tipos de mensaje que implican que DimRider
// puede haber cambiado (perfil, solicitud, revisión, activación/suspensión,
// contacto).
var riderSyncEventTypes = map[string]bool{
	"rider_updated": true,
}

// userSyncEventTypes son los tipos de mensaje que implican que DimUser puede
// haber cambiado. Lo publican tanto users/app (datos de la cuenta, bloqueo,
// rol Staff) como riders/app y shops/app (rol Rider/Owner, que también muta
// users.user_roles — ver "Trips & Riders" en CLAUDE.md para el precedente de
// un mismo evento publicado desde varios dominios).
var userSyncEventTypes = map[string]bool{
	"user_updated": true,
}

// shipmentSyncEventTypes son los tipos de mensaje del stream de shipments que
// implican que CubeShipment puede haber cambiado. shipment_status_changed lo
// publican tanto shipments/app (RequestShipment, CancelShipment) como
// riders/app (accept_shipment, advance_shipment) — mismo precedente que
// tripSyncEventTypes. shipment_no_driver también cambia el status a no_driver.
var shipmentSyncEventTypes = map[string]bool{
	"shipment_created":        true,
	"shipment_status_changed": true,
	"shipment_no_driver":      true,
}

// startOrderEventConsumer lee el stream con su propio consumer group y
// resincroniza puntualmente la entidad de cada evento relevante contra OLAP.
func startOrderEventConsumer(ctx context.Context, cfg config.ETLConfig, etlApp ports.ETLApp) {
	consumer := redisStream.NewConsumerRedisStreamConnection(ctx, cfg.Redis.URL, cfg.RedisStream)

	for {
		messages, err := consumer.Consume(ctx)
		if err != nil {
			slog.Error("etl consumer: error consumiendo mensajes", "error", err)
			continue
		}

		for _, msg := range messages {
			if ok, err := consumer.StillMine(ctx, msg.ID); err != nil {
				slog.Error("etl consumer: error en still mine", "msg_id", msg.ID, "error", err)
				continue
			} else if !ok {
				slog.Error("etl consumer: mensaje ya no es mío", "msg_id", msg.ID)
			}

			if err := processStreamEvent(ctx, msg, etlApp); err != nil {
				// Una entidad que ya no existe (borrada, condición de carrera
				// con el evento) no tiene nada que resincronizar: reintentarla
				// para siempre solo atasca el consumer group con el mismo
				// mensaje (ver el fix equivalente en cmd/notifications).
				if !errors.Is(err, errcustom.ErrNotFound) {
					slog.Error("etl consumer: error procesando evento", "msg_id", msg.ID, "error", err)
					continue
				}
				slog.Warn("etl consumer: entidad no encontrada, se descarta", "msg_id", msg.ID, "error", err)
			}

			if err := consumer.Ack(ctx, msg.ID); err != nil {
				slog.Error("etl consumer: error al confirmar mensaje", "msg_id", msg.ID, "error", err)
			}
		}
	}
}

// processStreamEvent despacha por tipo de mensaje a la entidad que le
// corresponde. Un tipo que no es de ninguna entidad sincronizada por evento
// (p.ej. withdrawal_requested, trip_created) se confirma sin procesar.
func processStreamEvent(ctx context.Context, msg redis.XMessage, etlApp ports.ETLApp) error {
	messageType, _ := msg.Values["type"].(string)
	switch {
	case orderSyncEventTypes[messageType]:
		return processOrderEvent(ctx, msg, messageType, etlApp)
	case fareQuoteSyncEventTypes[messageType]:
		return processFareQuoteEvent(ctx, msg, messageType, etlApp)
	case deliveryQuoteSyncEventTypes[messageType]:
		return processDeliveryQuoteEvent(ctx, msg, messageType, etlApp)
	case tripSyncEventTypes[messageType]:
		return processTripEvent(ctx, msg, messageType, etlApp)
	case shopSyncEventTypes[messageType]:
		return processShopEvent(ctx, msg, messageType, etlApp)
	case riderSyncEventTypes[messageType]:
		return processRiderEvent(ctx, msg, messageType, etlApp)
	case userSyncEventTypes[messageType]:
		return processUserEvent(ctx, msg, messageType, etlApp)
	case shipmentSyncEventTypes[messageType]:
		return processShipmentEvent(ctx, msg, messageType, etlApp)
	default:
		return nil
	}
}

func processOrderEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	orderID, ok := parseEventID(msg, messageType, "order_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncOrderByID(ctx, orderID); err != nil {
		return err
	}
	slog.Info("etl consumer: orden resincronizada", "order_id", orderID, "type", messageType)
	return nil
}

func processFareQuoteEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	quoteID, ok := parseEventID(msg, messageType, "quote_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncFareQuoteByID(ctx, quoteID); err != nil {
		return err
	}
	slog.Info("etl consumer: cotización resincronizada", "quote_id", quoteID, "type", messageType)
	return nil
}

func processDeliveryQuoteEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	quoteID, ok := parseEventID(msg, messageType, "quote_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncDeliveryQuoteByID(ctx, quoteID); err != nil {
		return err
	}
	slog.Info("etl consumer: cotización de delivery resincronizada", "quote_id", quoteID, "type", messageType)
	return nil
}

func processTripEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	tripID, ok := parseEventID(msg, messageType, "trip_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncTripByID(ctx, tripID); err != nil {
		return err
	}
	slog.Info("etl consumer: viaje resincronizado", "trip_id", tripID, "type", messageType)
	return nil
}

func processShopEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	shopID, ok := parseEventID(msg, messageType, "shop_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncShopByID(ctx, shopID); err != nil {
		return err
	}
	slog.Info("etl consumer: local resincronizado", "shop_id", shopID, "type", messageType)
	return nil
}

func processRiderEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	riderID, ok := parseEventID(msg, messageType, "rider_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncRiderByID(ctx, riderID); err != nil {
		return err
	}
	slog.Info("etl consumer: conductor resincronizado", "rider_id", riderID, "type", messageType)
	return nil
}

func processUserEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	userID, ok := parseEventID(msg, messageType, "user_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncUserByID(ctx, userID); err != nil {
		return err
	}
	slog.Info("etl consumer: cuenta resincronizada", "user_id", userID, "type", messageType)
	return nil
}

func processShipmentEvent(ctx context.Context, msg redis.XMessage, messageType string, etlApp ports.ETLApp) error {
	shipmentID, ok := parseEventID(msg, messageType, "shipment_id")
	if !ok {
		return nil
	}

	if err := etlApp.SyncShipmentByID(ctx, shipmentID); err != nil {
		return err
	}
	slog.Info("etl consumer: envío resincronizado", "shipment_id", shipmentID, "type", messageType)
	return nil
}

// parseEventID extrae y parsea el campo de ID del payload. Un campo ausente
// o inválido se descarta con warning (no se reintenta indefinidamente): un
// payload malformado no se arregla solo reintentándolo.
func parseEventID(msg redis.XMessage, messageType, field string) (uuid.UUID, bool) {
	raw, ok := msg.Values[field].(string)
	if !ok || raw == "" {
		slog.Warn("etl consumer: evento sin "+field+" válido, se descarta", "msg_id", msg.ID, "type", messageType)
		return uuid.UUID{}, false
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		slog.Warn("etl consumer: "+field+" inválido, se descarta", "msg_id", msg.ID, field, raw)
		return uuid.UUID{}, false
	}
	return id, true
}
