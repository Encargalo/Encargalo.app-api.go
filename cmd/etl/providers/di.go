package providers

import (
	"Encargalo.app-api.go/internal/analytics/app"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	"Encargalo.app-api.go/internal/analytics/repo"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/labstack/echo/v4"
	"go.uber.org/dig"
)

func BuildContainer() *dig.Container {
	c := dig.New()

	_ = c.Provide(func() config.Config {
		config.Environments()
		return *config.Get()
	})

	_ = c.Provide(func() *echo.Echo {
		e := echo.New()
		e.HTTPErrorHandler = errcustom.HTTPErrorHandler
		return e
	})

	_ = c.Provide(postgres.NewOLTPConnection)
	_ = c.Provide(postgres.NewOLAPConnection)
	_ = c.Provide(repo.NewOLTPSourceRepo)
	_ = c.Provide(repo.NewOLAPSinkRepo)
	_ = c.Provide(repo.NewSyncLogRepo)
	_ = c.Provide(func(
		source ports.OLTPSourceRepo,
		sink ports.OLAPSinkRepo,
		syncLog ports.SyncLogRepo,
		cfg config.Config,
	) ports.ETLApp {
		return app.NewETLApp(source, sink, syncLog, app.Lookbacks{
			Default: cfg.ETL.DefaultLookback(),
			Initial: cfg.ETL.InitialLookback(),
		})
	})

	return c
}
