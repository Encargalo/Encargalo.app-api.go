package config

import (
	"log"
	"os"
	"strings"
	"sync"

	gosuite "github.com/shepherd-go/gosuite/config"
)

var (
	notificationsOnce sync.Once
	notificationsCfg  *NotificationsConfig
)

// NotificationsConfig es la configuración del consumer cmd/notifications
// (sin JWT/Google/Cloudinary, con los destinos de notificación y Grafana).
type NotificationsConfig struct {
	Server        Server                  `env:"server"`
	Environment   Environment             `env:"environment"`
	Postgres      Postgres                `env:"postgres"`
	Redis         Redis                   `env:"redis"`
	RedisStream   NotificationsRedisGroup `env:"redis_stream"`
	Discord       Discord                 `env:"discord"`
	Slack         Slack                   `env:"slack"`
	Observability Observability           `env:"observability"`
	Grafana       Grafana                 `env:"grafana"`
}

// NotificationsRedisGroup agrega al stream el grupo y consumer que solo
// necesita el lado consumidor (la API únicamente produce).
type NotificationsRedisGroup struct {
	StreamName   string `validate:"required" env:"stream_name"`
	GroupName    string `validate:"required" env:"group_name"`
	ConsumerName string `validate:"required" env:"consumer_name"`
	Db           int    `validate:"min=0" env:"db"`
}

type Discord struct {
	// Enabled is opt-in. Discord is off unless DISCORD_ENABLED is true.
	Enabled                   string `env:"enabled"`
	Url                       string `env:"url"`
	Path                      string `env:"path"`
	OrdersStatusID            string `env:"orders_status_id"`
	OrdersRejectedID          string `env:"orders_rejected_id"`
	OrdersPendingUnassignedID string `env:"orders_pending_unassigned_id"`
	OrdersAutoRejectedID      string `env:"orders_auto_rejected_id"`
	PaymentProofsID           string `env:"payment_proofs_id"`
	BalanceRefundsID          string `env:"balance_refunds_id"`
	ErrorsID                  string `env:"errors_id"`
	DevelopChannelID          string `env:"dev_channel_id"`
}

func (d Discord) IsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(d.Enabled)) {
	case "true", "1", "yes", "on":
		return d.Url != "" && d.Path != ""
	default:
		return false
	}
}

// Slack holds Incoming Webhook settings. All fields are optional until Slack
// is wired in NewFanout. Same as Discord: POST {url}{path}{id} per Kind.
type Slack struct {
	Url                       string `env:"url"`
	Path                      string `env:"path"`
	OrdersStatusID            string `env:"orders_status_id"`
	OrdersRejectedID          string `env:"orders_rejected_id"`
	OrdersPendingUnassignedID string `env:"orders_pending_unassigned_id"`
	OrdersAutoRejectedID      string `env:"orders_auto_rejected_id"`
	PaymentProofsID           string `env:"payment_proofs_id"`
	BalanceRefundsID          string `env:"balance_refunds_id"`
	ErrorsID                  string `env:"errors_id"`
	DevelopChannelID          string `env:"dev_channel_id"`
}

func (s Slack) Configured() bool {
	return s.Url != "" && s.Path != ""
}

type Grafana struct {
	WebhookSecret string `validate:"required" env:"webhook_secret"`
}

func GetNotifications() *NotificationsConfig {
	if notificationsCfg == nil {
		log.Panic("Notifications configuration has not yet been initialized")
	}
	return notificationsCfg
}

func NotificationsEnvironments() {
	notificationsOnce.Do(func() {
		// Railway inyecta PORT; nuestro tag espera SERVER_PORT.
		if os.Getenv("SERVER_PORT") == "" {
			if p := os.Getenv("PORT"); p != "" {
				_ = os.Setenv("SERVER_PORT", p)
			}
		}

		notificationsCfg = new(NotificationsConfig)
		if err := gosuite.GetConfigFromEnv(notificationsCfg); err != nil {
			log.Panicf("error parsing notifications environment vars \n%v", err)
		}
	})
}
