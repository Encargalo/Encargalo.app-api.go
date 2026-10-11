package models_test

import (
	"testing"
	"time"

	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestTrip_ToDTO_ExposesStartedAt(t *testing.T) {
	started := time.Date(2026, 10, 6, 15, 42, 0, 0, time.UTC)
	trip := tripsModels.Trip{Status: tripsModels.TripStatusInProgress, StartedAt: &started}

	assert.Equal(t, &started, trip.ToDTO().StartedAt)
	assert.Equal(t, &started, trip.ToCustomerDTO().StartedAt)
}

func TestTrip_ToDTO_OmitsStartedAtBeforePickup(t *testing.T) {
	trip := tripsModels.Trip{Status: tripsModels.TripStatusArriving}

	assert.Nil(t, trip.ToDTO().StartedAt)
}

func TestTrip_ToDTO_CashTripCollectsFullFareInBsAtFrozenRate(t *testing.T) {
	fare, rate := 2.5, 40.0
	trip := tripsModels.Trip{MethodPayment: tripsModels.MethodEfectivo, Fare: &fare, BcvRate: &rate}

	resp := trip.ToDTO()

	assert.Equal(t, 2.5, *resp.CollectUSD)
	assert.Equal(t, 100.0, *resp.CollectBs)
}

func TestTrip_ToDTO_SplitTripCollectsOnlyCashPartInBs(t *testing.T) {
	fare, rate, cashBs := 2.5, 40.0, 60.0
	trip := tripsModels.Trip{MethodPayment: tripsModels.MethodPagoMovil, Fare: &fare, BcvRate: &rate, CashAmountBs: &cashBs}

	resp := trip.ToDTO()

	assert.Equal(t, 1.5, *resp.CollectUSD)
	assert.Equal(t, 60.0, *resp.CollectBs)
}

func TestTrip_ToDTO_NoCollectBsWhenNothingToCollect(t *testing.T) {
	fare, rate := 2.5, 40.0
	trip := tripsModels.Trip{MethodPayment: tripsModels.MethodPagoMovil, Fare: &fare, BcvRate: &rate}

	assert.Nil(t, trip.ToDTO().CollectBs)
}

func TestTrip_ApplyFrozenBsFare_UsesQuoteRate(t *testing.T) {
	fare, rate := 2.5, 40.0
	trip := tripsModels.Trip{Fare: &fare, BcvRate: &rate}

	trip.ApplyFrozenBsFare()

	assert.Equal(t, 100.0, *trip.FareBs)
}

func TestTrip_ApplyFrozenBsFare_NoRateLeavesFareBsEmpty(t *testing.T) {
	fare := 2.5
	trip := tripsModels.Trip{Fare: &fare}

	trip.ApplyFrozenBsFare()

	assert.Nil(t, trip.FareBs)
}
