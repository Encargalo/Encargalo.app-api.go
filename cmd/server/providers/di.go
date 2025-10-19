package providers

import (
	"Encargalo.app-api.go/api/router"
	"Encargalo.app-api.go/api/router/groups"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/adapters/redis"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/jwt"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/handler"
	"Encargalo.app-api.go/internal/shops/repo"
	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	appAuth "Encargalo.app-api.go/internal/auth/app"
	handAuth "Encargalo.app-api.go/internal/auth/handler"
	repoAuth "Encargalo.app-api.go/internal/auth/repo"

	appProducts "Encargalo.app-api.go/internal/products/app"
	handProducts "Encargalo.app-api.go/internal/products/handler"
	repoProducts "Encargalo.app-api.go/internal/products/repo"

	appCustomer "Encargalo.app-api.go/internal/customers/app"
	repoCustomer "Encargalo.app-api.go/internal/customers/repo"
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
	_ = Container.Provide(redis.NewRedisConnection)

	_ = Container.Provide(router.New)

	_ = Container.Provide(groups.NewAuthGroup)
	_ = Container.Provide(groups.NewShopsGroup)
	_ = Container.Provide(groups.NewProductsGroup)

	_ = Container.Provide(handAuth.NewAuthHandler)
	_ = Container.Provide(handler.NewShopsHandler)
	_ = Container.Provide(handProducts.NewProducsHandler)

	_ = Container.Provide(appAuth.NewAuthApp)
	_ = Container.Provide(app.NewShopsApp)
	_ = Container.Provide(appProducts.NewProductsApp)
	_ = Container.Provide(appCustomer.NewCustomerApp)

	_ = Container.Provide(repoAuth.NewAuthRepo)
	_ = Container.Provide(repo.NewShopsRepository)
	_ = Container.Provide(repoProducts.NewProductsRepo)
	_ = Container.Provide(repoCustomer.NewCustomersRepo)

	_ = Container.Provide(bycript.NewHashPassword)
	_ = Container.Provide(jwt.NewSessionUtils)
	_ = Container.Provide(cookie.NewCookie)

	return Container

}
