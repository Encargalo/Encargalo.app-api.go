package providers

import (
	"Encargalo.app-api.go/internal/notifications/app"
	"Encargalo.app-api.go/internal/notifications/handler"
	"Encargalo.app-api.go/internal/notifications/notifier"
	"Encargalo.app-api.go/internal/notifications/repo"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/config"
	"go.uber.org/dig"
)

func BuildContainer() *dig.Container {
	c := dig.New()

	_ = c.Provide(func() config.NotificationsConfig {
		config.NotificationsEnvironments()
		return *config.GetNotifications()
	})

	_ = c.Provide(postgres.NewNotificationsConnection)

	_ = c.Provide(handler.NewNotification)
	_ = c.Provide(handler.NewGrafanaWebhook)

	_ = c.Provide(app.NewNotifications)
	_ = c.Provide(notifier.NewFanout)

	_ = c.Provide(repo.NewRepository)

	return c
}
