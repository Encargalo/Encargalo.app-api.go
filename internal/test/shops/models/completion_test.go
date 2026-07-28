package models_test

import (
	"testing"

	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/stretchr/testify/assert"
)

func TestShopCompletion_Empty(t *testing.T) {
	shop := &shopsmodels.Shop{Name: "Pizzería Don Luigi", Type: "Restaurant"}

	comp := shop.Completion()

	assert.False(t, comp.IsComplete)
	assert.Equal(t, 0, comp.FilledCount)
	assert.Equal(t, 6, comp.RequiredCount)
	assert.ElementsMatch(t, []string{"slug", "logo_image", "banner", "address", "home_phone", "coordinates"}, comp.Missing)
}

func TestShopCompletion_Partial(t *testing.T) {
	lat, lon := 10.4806, -66.9036
	shop := &shopsmodels.Shop{
		Name:      "Pizzería Don Luigi",
		Type:      "Restaurant",
		LogoImage: "https://cdn/logo.png",
		Latitude:  &lat,
		Longitude: &lon,
	}

	comp := shop.Completion()

	assert.False(t, comp.IsComplete)
	assert.Equal(t, 2, comp.FilledCount)
	assert.ElementsMatch(t, []string{"slug", "banner", "address", "home_phone"}, comp.Missing)
}

func TestShopCompletion_Complete(t *testing.T) {
	lat, lon := 10.4806, -66.9036
	shop := &shopsmodels.Shop{
		Name:      "Pizzería Don Luigi",
		Type:      "Restaurant",
		Slug:      "don-luigi",
		LogoImage: "https://cdn/logo.png",
		Banner:    "https://cdn/banner.png",
		Address:   "Av. Principal, local 3",
		HomePhone: "+584121234567",
		Latitude:  &lat,
		Longitude: &lon,
	}

	comp := shop.Completion()

	assert.True(t, comp.IsComplete)
	assert.Empty(t, comp.Missing)
	assert.Equal(t, comp.RequiredCount, comp.FilledCount)
}
