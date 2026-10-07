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
