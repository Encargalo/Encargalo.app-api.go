package providers

import (
	"Encargalo.app-api.go/api/router"
	"Encargalo.app-api.go/api/router/groups"
	authMiddle "Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/pkg/logs"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/adapters/googleauth"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/adapters/redis"
	"Encargalo.app-api.go/internal/shared/adapters/slack"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	appShops "Encargalo.app-api.go/internal/shops/app"
	shopsPorts "Encargalo.app-api.go/internal/shops/domain/ports"
	handShops "Encargalo.app-api.go/internal/shops/handler"
	repoShops "Encargalo.app-api.go/internal/shops/repo"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	appAuth "Encargalo.app-api.go/internal/auth/app"
	handAuth "Encargalo.app-api.go/internal/auth/handler"
	repoAuth "Encargalo.app-api.go/internal/auth/repo"

	appProducts "Encargalo.app-api.go/internal/products/app"
	productsPorts "Encargalo.app-api.go/internal/products/domain/ports"
	handProducts "Encargalo.app-api.go/internal/products/handler"
	repoProducts "Encargalo.app-api.go/internal/products/repo"

	appCustomer "Encargalo.app-api.go/internal/users/app"
	handCustomer "Encargalo.app-api.go/internal/users/handler"
	repoCustomer "Encargalo.app-api.go/internal/users/repo"

	appOrders "Encargalo.app-api.go/internal/orders/app"
	handOrders "Encargalo.app-api.go/internal/orders/handler"
	repoOrders "Encargalo.app-api.go/internal/orders/repo"

	appRiders "Encargalo.app-api.go/internal/riders/app"
	handRiders "Encargalo.app-api.go/internal/riders/handler"
	repoRiders "Encargalo.app-api.go/internal/riders/repo"

	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	usersPorts "Encargalo.app-api.go/internal/users/domain/ports"

	appEmployee "Encargalo.app-api.go/internal/employee/app"
	repoEmployee "Encargalo.app-api.go/internal/employee/repo"

	appBcv "Encargalo.app-api.go/internal/bcv/app"
	repoBcv "Encargalo.app-api.go/internal/bcv/repo"

	appPayment "Encargalo.app-api.go/internal/payment_methods/app"
	handPayment "Encargalo.app-api.go/internal/payment_methods/handler"
	repoPayment "Encargalo.app-api.go/internal/payment_methods/repo"
)

var Container *dig.Container

func BuildContainer() *dig.Container {
	Container = dig.New()

	_ = Container.Provide(func() config.Config {
		config.Environments()
		return *config.Get()
	})

	_ = Container.Provide(func() *echo.Echo {
		e := echo.New()
		e.HTTPErrorHandler = errcustom.HTTPErrorHandler
		return e
	})

	_ = Container.Provide(func() *cloudinary.Cloudinary {
		cld, err := cloudinary.New()
		if err != nil {
			panic(err)
		}

		return cld
	})

	_ = Container.Provide(postgres.NewPostgresConnection)
	_ = Container.Provide(redis.NewRedisConnection)
	_ = Container.Provide(redis.NewProducerRedisStreamConnection)
	_ = Container.Provide(cache.NewCache)

	_ = Container.Provide(router.New)

	_ = Container.Provide(groups.NewAuthGroup)
	_ = Container.Provide(groups.NewShopsGroup)
	_ = Container.Provide(groups.NewProductsGroup)
	_ = Container.Provide(groups.NewUsersGroup)
	_ = Container.Provide(groups.NewOrdersGroup)
	_ = Container.Provide(groups.NewPaymentMethodsGroup)
	_ = Container.Provide(handPayment.NewPaymentMethodsHandler)
	_ = Container.Provide(appPayment.NewPaymentMethodsApp)
	_ = Container.Provide(repoPayment.NewPaymentMethodsRepo)

	_ = Container.Provide(requestinfo.NewRequestMiddleware)
	_ = Container.Provide(sharedmw.NewRateLimiter)
	_ = Container.Provide(authMiddle.NewAuthMidlleware)
	_ = Container.Provide(handAuth.NewAuthHandler)
	_ = Container.Provide(handCustomer.NewUsersHandler)
	_ = Container.Provide(handShops.NewShopsHandler)
	_ = Container.Provide(handShops.NewShopFinanceHandler)
	_ = Container.Provide(handShops.NewShopStatsHandler)
	_ = Container.Provide(repoShops.NewShopFinanceRepo)
	_ = Container.Provide(repoShops.NewShopStatsRepo)
	_ = Container.Provide(appShops.NewShopFinanceApp)
	_ = Container.Provide(appShops.NewShopStatsApp)
	_ = Container.Provide(func(fa shopsPorts.ShopFinanceApp) ordersPorts.ShopCommissionRecorder {
		return fa.(ordersPorts.ShopCommissionRecorder)
	})
	_ = Container.Provide(func(mgr ordersPorts.AppManagerOrder) shopsPorts.ShopStatusBroadcaster {
		return mgr
	})
	// ShopsApp ya declara GetLaunchPromoEndsAt con la misma firma que
	// ShopLaunchPromoProvider, así que esta es una conversión de interfaz
	// implícita verificada en tiempo de compilación (no un type assertion en
	// runtime): si algún día las firmas divergen, esto deja de compilar aquí
	// mismo en vez de fallar al construir el contenedor DI.
	_ = Container.Provide(func(sa shopsPorts.ShopsApp) productsPorts.ShopLaunchPromoProvider {
		return sa
	})
	_ = Container.Provide(handProducts.NewProducsHandler)
	_ = Container.Provide(handCustomer.NewUsersAddressHandler)
	_ = Container.Provide(handOrders.NewOrderHandler)
	_ = Container.Provide(handOrders.New)
	_ = Container.Provide(handRiders.New)
	_ = Container.Provide(handRiders.NewFinanceHandler)
	_ = Container.Provide(handRiders.NewStaffRiderHandler)
	_ = Container.Provide(repoRiders.NewRidersRepo)
	_ = Container.Provide(repoRiders.NewFinanceRepo)
	_ = Container.Provide(repoRiders.NewRidersStream)
	_ = Container.Provide(repoRiders.NewRiderProfileRepo)
	_ = Container.Provide(appRiders.NewRiderProfileApp)
	_ = Container.Provide(func(mgr ridersPorts.RidersApp) ordersPorts.OrderRiderBroadcaster {
		return mgr.(ordersPorts.OrderRiderBroadcaster)
	})
	_ = Container.Provide(func(fa ridersPorts.FinanceApp) ordersPorts.RideEarningRecorder {
		return fa.(ordersPorts.RideEarningRecorder)
	})

	_ = Container.Provide(appCustomer.NewUserApp)
	_ = Container.Provide(appCustomer.NewStaffRefundHub)
	_ = Container.Provide(func(h *appCustomer.StaffRefundHubImpl) usersPorts.StaffRefundHub {
		return h
	})
	_ = Container.Provide(func(s ordersPorts.RedisStream) usersPorts.StreamPublisher {
		return s
	})
	_ = Container.Provide(appCustomer.NewCustomerFinanceApp)
	_ = Container.Provide(repoCustomer.NewCustomerFinanceRepo)
	_ = Container.Provide(handCustomer.NewCustomerFinanceHandler)
	_ = Container.Provide(handCustomer.NewStaffRefundHandler)
	_ = Container.Provide(appCustomer.NewRefundAlertWorker)
	_ = Container.Provide(appAuth.NewAuthApp)
	_ = Container.Provide(appShops.NewShopsApp)
	_ = Container.Provide(appShops.NewBatchCloseWorker)
	_ = Container.Provide(appProducts.NewProductsApp)
	_ = Container.Provide(appCustomer.NewUsersAddressApp)
	_ = Container.Provide(appOrders.NewStaffReviewHub)
	_ = Container.Provide(func(h *appOrders.StaffReviewHubImpl) ordersPorts.StaffReviewHub {
		return h
	})
	_ = Container.Provide(repoOrders.NewPaymentReviewsRepo)
	_ = Container.Provide(func(fa usersPorts.CustomerFinanceApp) ordersPorts.CustomerBalanceCreditor {
		return fa
	})
	_ = Container.Provide(appOrders.NewPaymentReviewsApp)
	_ = Container.Provide(handOrders.NewStaffReviewsHandler)
	_ = Container.Provide(appOrders.NewOrderApp)
	_ = Container.Provide(appOrders.NewManagerOrderApp)
	_ = Container.Provide(appOrders.NewPendingAcceptWorker)
	_ = Container.Provide(appRiders.New)
	_ = Container.Provide(appRiders.NewFinanceApp)

	_ = Container.Provide(repoAuth.NewAuthRepo)
	_ = Container.Provide(repoShops.NewShopsRepository)
	_ = Container.Provide(repoProducts.NewProductsRepo)
	_ = Container.Provide(repoCustomer.NewUsersRepo)
	_ = Container.Provide(repoCustomer.NewUsersAddressRepo)
	_ = Container.Provide(repoOrders.NewOrdersRepo)
	_ = Container.Provide(repoOrders.NewOrderQuotesRepo)
	_ = Container.Provide(repoOrders.NewRepositoryProducerStream)

	_ = Container.Provide(appEmployee.New)
	_ = Container.Provide(repoEmployee.New)

	_ = Container.Provide(appBcv.New)
	_ = Container.Provide(repoBcv.NewScraper)
	_ = Container.Provide(repoBcv.NewCacheRepo)
	_ = Container.Provide(repoBcv.NewCalendarScraper)
	_ = Container.Provide(repoBcv.NewHolidayRepo)

	_ = Container.Provide(googleauth.NewVerifier)
	_ = Container.Provide(bycript.NewHashPassword)
	_ = Container.Provide(jwt.NewSessionUtils)
	_ = Container.Provide(slack.NewConnectionSlack)
	_ = Container.Provide(logs.NewLogs)
	_ = Container.Provide(cookie.NewCookieFactory)
	_ = Container.Provide(uploadimage.NewUploadImage)

	return Container

}
