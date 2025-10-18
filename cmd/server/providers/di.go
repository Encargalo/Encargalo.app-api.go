package providers

import (
	"Encargalo.app-api.go/api/router"
	"Encargalo.app-api.go/api/router/groups"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/handler"
	"Encargalo.app-api.go/internal/shops/repo"
	"github.com/labstack/echo/v4"
	"go.uber.org/dig"
)

var Container *dig.Container

func BuildContainer() *dig.Container {
	Container = dig.New()

	_ = Container.Provide(func() config.Config {
		config.Environments()
		return *config.Get()
	})

	_ = Container.Provide(func() *echo.Echo {
		return echo.New()
	})

	_ = Container.Provide(postgres.NewPostgresConnection)

	_ = Container.Provide(router.New)

	_ = Container.Provide(groups.NewShopsGroup)

	_ = Container.Provide(handler.NewShopsHandler)

	_ = Container.Provide(app.NewShopsApp)

	_ = Container.Provide(repo.NewShopsRepository)

	return Container

}
