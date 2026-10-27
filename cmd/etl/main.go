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
				slog.Error("etl consumer: error procesando evento", "msg_id", msg.ID, "error", err)
				continue
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
