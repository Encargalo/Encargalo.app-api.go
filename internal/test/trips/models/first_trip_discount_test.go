package models_test

import (
	"testing"

	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestFirstTripDiscountUSD_HalfOfFare(t *testing.T) {
	assert.Equal(t, 4.0, tripsModels.FirstTripDiscountUSD(8))
}

func TestFirstTripDiscountUSD_CappedAtSixDollars(t *testing.T) {
	assert.Equal(t, 6.0, tripsModels.FirstTripDiscountUSD(20))
}

func TestFirstTripDiscountUSD_ZeroFare(t *testing.T) {
	assert.Equal(t, 0.0, tripsModels.FirstTripDiscountUSD(0))
}

func TestTrip_GrossFareUSD_AddsDiscountBack(t *testing.T) {
	fare := 4.0
	trip := tripsModels.Trip{Fare: &fare, DiscountUSD: 4}

	assert.Equal(t, 8.0, trip.GrossFareUSD())
	assert.Equal(t, 4.0, trip.ToDTO().DiscountUSD)
}
