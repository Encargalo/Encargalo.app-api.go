package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"Encargalo.app-api.go/cmd/notifications/providers"
	"Encargalo.app-api.go/internal/notifications/handler"
	redisStream "Encargalo.app-api.go/internal/shared/adapters/redis"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/otelsetup"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/redis/go-redis/v9"
)

func main() {
	config.NotificationsEnvironments()
	cfg := config.GetNotifications()

	ctx := context.Background()
	logHandler, shutdownTelemetry, err := otelsetup.Setup(ctx, cfg.Observability, cfg.Environment.Current, reqctx.ServiceNameNotifications)
	if err != nil {
		slog.Error("fallo al iniciar telemetría", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("fallo al cerrar telemetría", "error", err.Error())
		}
	}()

	applog.SetupJSON(slog.LevelInfo, logHandler, reqctx.ServiceNameNotifications)

	container := providers.BuildContainer()
	if err := container.Invoke(func(notificationHand handler.Notification, grafanaWebhookHand handler.GrafanaWebhook) {
		consumer := redisStream.NewConsumerRedisStreamConnection(ctx, *cfg)

		go startHTTPServer(cfg.Server.Port, grafanaWebhookHand)

		for {
			message, err := consumer.Consume(ctx)
			if err != nil {
				slog.Error("error consumiendo mensajes", "error", err)
				continue
			}

			if len(message) > 0 {
				slog.Info("mensajes pendientes", "count", len(message))
			}

			for _, msg := range message {
				if ok, err := consumer.StillMine(ctx, msg.ID); err != nil {
					slog.Error("error en still mine", "msg_id", msg.ID, "error", err)
					continue
				} else if !ok {
					slog.Error("mensaje ya no es mío", "msg_id", msg.ID)
				}

				if err := processMessage(msg, notificationHand); err != nil {
					continue
				}

				if err = consumer.Ack(ctx, msg.ID); err != nil {
					slog.Error("error al confirmar mensaje", "msg_id", msg.ID, "error", err)
					continue
				}
			}
		}
	}); err != nil {
		panic(err)
	}
}

func startHTTPServer(port int, grafanaWebhookHand handler.GrafanaWebhook) {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhooks/grafana", grafanaWebhookHand.Handle)

	addr := fmt.Sprintf(":%d", port)
	slog.Info("servidor HTTP escuchando", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("error en servidor HTTP", "error", err)
	}
}

func processMessage(msg redis.XMessage, notificationHand handler.Notification) error {
	messageType, ok := msg.Values["type"].(string)
	if !ok {
		slog.Warn("mensaje sin campo type válido, se descarta", "msg_id", msg.ID, "value", msg.Values["type"])
		return errors.New("message type not recognized")
	}

	switch messageType {
	case "order_created":
		if err := notificationHand.OrderCreated(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "order_completed":
		if err := notificationHand.OrderCompleted(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "order_rejected":
		if err := notificationHand.OrderRejected(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "order_pending_unassigned":
		if err := notificationHand.OrderPendingUnassigned(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "order_auto_rejected":
		if err := notificationHand.OrderAutoRejected(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "order_no_rider":
		if err := notificationHand.OrderNoRider(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "payment_proof_pending":
		if err := notificationHand.PaymentProofPending(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "balance_refund_pending":
		if err := notificationHand.BalanceRefundPending(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "withdrawal_requested":
		if err := notificationHand.WithdrawalRequested(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "trip_created":
		if err := notificationHand.TripCreated(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	case "trip_no_driver":
		if err := notificationHand.TripNoDriver(msg); err != nil {
			slog.Error("error de notificación", "msg_id", msg.ID, "error", err)
			return err
		}
	default:
		slog.Warn("tipo de mensaje no reconocido", "type", messageType)
		return errors.New("message type not recognized")
	}

	return nil
}
