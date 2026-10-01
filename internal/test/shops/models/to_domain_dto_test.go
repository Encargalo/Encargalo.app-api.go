package models_test

import (
	"testing"

	"Encargalo.app-api.go/internal/shops/domain/entity"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShopToDomainDTO_WithoutCoordinates(t *testing.T) {
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Local e2e", Type: "Restaurant"}

	assert.NotPanics(t, func() {
		dto := shop.ToDomainDTO(entity.ShopDistance{ShopID: shop.ID})
		assert.Zero(t, dto.Latitude)
		assert.Zero(t, dto.Longitude)
	})
}

func TestShopToDomainDTO_WithCoordinates(t *testing.T) {
	lat, lon := 10.4806, -66.9036
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Pizzería Don Luigi", Type: "Restaurant", Latitude: &lat, Longitude: &lon}

	dto := shop.ToDomainDTO(entity.ShopDistance{ShopID: shop.ID})

	assert.Equal(t, lat, dto.Latitude)
	assert.Equal(t, lon, dto.Longitude)
}

func TestShopsToDomainDTO_MixedCoordinates(t *testing.T) {
	lat, lon := 10.4806, -66.9036
	withCoords := shopsmodels.Shop{ID: uuid.New(), Type: "Restaurant", Latitude: &lat, Longitude: &lon}
	withoutCoords := shopsmodels.Shop{ID: uuid.New(), Type: "Restaurant"}
	list := shopsmodels.Shops{withCoords, withoutCoords}

	resp := list.ToDomainDTO([]entity.ShopDistance{{ShopID: withCoords.ID}, {ShopID: withoutCoords.ID}})

	assert.Len(t, resp, 2)
}
