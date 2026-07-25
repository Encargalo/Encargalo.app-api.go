package app_test

import (
	"testing"

	"Encargalo.app-api.go/internal/orders/app"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestValidateTurboCoverage(t *testing.T) {
	turboShop := &models.ShopOrder{Type: "Turbo", Latitude: 10.4806, Longitude: -66.9036}
	restaurantShop := &models.ShopOrder{Type: "Restaurant", Latitude: 10.4806, Longitude: -66.9036}

	cases := []struct {
		name    string
		shop    *models.ShopOrder
		address dtos.Address
		wantErr bool
	}{
		{"turbo dentro de 2km", turboShop, dtos.Address{Latitude: 10.4900, Longitude: -66.9036}, false},
		{"turbo fuera de 2km", turboShop, dtos.Address{Latitude: 10.6000, Longitude: -66.9036}, true},
		{"shop no turbo ignora la cobertura", restaurantShop, dtos.Address{Latitude: 20.0, Longitude: -60.0}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := app.ValidateTurboCoverage(tc.shop, tc.address)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
