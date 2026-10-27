package app_test

import (
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockordersports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	mockuploadimage "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mockproductsports "Encargalo.app-api.go/internal/mocks/products/domain/ports"
	mockshopsports "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	mockusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"github.com/stretchr/testify/mock"
)

// orderAppMocks agrupa las 14 dependencias de orderApp para no repetir el
// wiring en cada test (ver NewOrderApp en internal/orders/app/order.go).
type orderAppMocks struct {
	repo            *mockordersports.OrdersRepo
	shops           *mockshopsports.ShopsRepo
	users           *mockusersports.UsersRepo
	stream          *mockordersports.RedisStream
	upload          *mockuploadimage.UploadImage
	riderMgr        *mockordersports.OrderRiderBroadcaster
	finance         *mockordersports.RideEarningRecorder
	shopCommission  *mockordersports.ShopCommissionRecorder
	bcv             *mockbcvports.BcvApp
	reviewsRepo     *mockordersports.PaymentReviewsRepo
	staffHub        *mockordersports.StaffReviewHub
	customerBalance *mockordersports.CustomerBalanceCreditor
	quotes          *mockordersports.OrderQuotesRepo
	products        *mockproductsports.ProductsRepo
	fareQuotes      *mockordersports.OrderFareQuotesRepo
}

func newOrderAppFixture(t *testing.T) (ports.OrdersApp, *orderAppMocks) {
	t.Helper()
	m := &orderAppMocks{
		repo:            mockordersports.NewOrdersRepo(t),
		shops:           mockshopsports.NewShopsRepo(t),
		users:           mockusersports.NewUsersRepo(t),
		stream:          mockordersports.NewRedisStream(t),
		upload:          mockuploadimage.NewUploadImage(t),
		riderMgr:        mockordersports.NewOrderRiderBroadcaster(t),
		finance:         mockordersports.NewRideEarningRecorder(t),
		shopCommission:  mockordersports.NewShopCommissionRecorder(t),
		bcv:             mockbcvports.NewBcvApp(t),
		reviewsRepo:     mockordersports.NewPaymentReviewsRepo(t),
		staffHub:        mockordersports.NewStaffReviewHub(t),
		customerBalance: mockordersports.NewCustomerBalanceCreditor(t),
		quotes:          mockordersports.NewOrderQuotesRepo(t),
		products:        mockproductsports.NewProductsRepo(t),
		fareQuotes:      mockordersports.NewOrderFareQuotesRepo(t),
	}
	// recordFareQuote es fire-and-forget (ver CreateQuote): no todos los tests
	// lo disparan, así que la expectativa es opcional.
	m.fareQuotes.On("Insert", mock.Anything, mock.Anything).Return(nil).Maybe()
	// recordFareQuote publica delivery_quote_created tras el Insert; mismo motivo.
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil).Maybe()
	svc := app.NewOrderApp(
		m.repo, m.shops, m.users, m.stream, m.upload, m.riderMgr, m.finance,
		m.shopCommission, m.bcv, m.reviewsRepo, m.staffHub, m.customerBalance,
		m.quotes, m.products, m.fareQuotes,
	)
	return svc, m
}
