package app_test

import (
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockuploadimage "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mocksproducts "Encargalo.app-api.go/internal/mocks/products/domain/ports"
	"Encargalo.app-api.go/internal/products/app"
	"Encargalo.app-api.go/internal/products/domain/ports"
)

// productAppMocks agrupa las 4 dependencias de productsApp (ver NewProductsApp
// en internal/products/app/products.go) para no repetir el wiring en cada test.
type productAppMocks struct {
	repo   *mocksproducts.ProductsRepo
	upload *mockuploadimage.UploadImage
	bcv    *mockbcvports.BcvApp
	promo  *mocksproducts.ShopLaunchPromoProvider
}

func newProductAppFixture(t *testing.T) (ports.ProductsApp, *productAppMocks) {
	t.Helper()
	m := &productAppMocks{
		repo:   mocksproducts.NewProductsRepo(t),
		upload: mockuploadimage.NewUploadImage(t),
		bcv:    mockbcvports.NewBcvApp(t),
		promo:  mocksproducts.NewShopLaunchPromoProvider(t),
	}
	svc := app.NewProductsApp(m.repo, m.upload, m.bcv, m.promo)
	return svc, m
}
