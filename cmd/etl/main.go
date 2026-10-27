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

// startOrderEventConsumer lee el stream con su propio consumer group y
// resincroniza puntualmente la orden de cada evento relevante contra OLAP.
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

			if err := processOrderEvent(ctx, msg, etlApp); err != nil {
				slog.Error("etl consumer: error procesando evento", "msg_id", msg.ID, "error", err)
				continue
			}

			if err := consumer.Ack(ctx, msg.ID); err != nil {
				slog.Error("etl consumer: error al confirmar mensaje", "msg_id", msg.ID, "error", err)
			}
		}
	}
}

func processOrderEvent(ctx context.Context, msg redis.XMessage, etlApp ports.ETLApp) error {
	messageType, _ := msg.Values["type"].(string)
	if !orderSyncEventTypes[messageType] {
		// No es un evento que afecte a CubeOrder (p.ej. withdrawal_requested,
		// trip_created): se confirma sin procesar, no es un fallo.
		return nil
	}

	orderIDStr, ok := msg.Values["order_id"].(string)
	if !ok || orderIDStr == "" {
		slog.Warn("etl consumer: evento sin order_id válido, se descarta", "msg_id", msg.ID, "type", messageType)
		return nil
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		slog.Warn("etl consumer: order_id inválido, se descarta", "msg_id", msg.ID, "order_id", orderIDStr)
		return nil
	}

	if err := etlApp.SyncOrderByID(ctx, orderID); err != nil {
		return err
	}
	slog.Info("etl consumer: orden resincronizada", "order_id", orderID, "type", messageType)
	return nil
}
