package models_test

import (
	"testing"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestRiderProfile_Completion_Incomplete(t *testing.T) {
	p := &models.RiderProfile{
		ProfilePhotoURL: new("https://x/a.jpg"),
		VehicleType:     new("moto"),
	}
	c := p.Completion()
	assert.False(t, c.IsComplete)
	assert.Contains(t, c.Missing, "vehicle_plate")
	assert.Contains(t, c.Missing, "id_photo")
	assert.Equal(t, 11, c.RequiredCount)
}

func TestRiderProfile_Completion_BikeNoPlate(t *testing.T) {
	p := &models.RiderProfile{
		ProfilePhotoURL:    new("a"),
		IDPhotoURL:         new("b"),
		CarnetPhotoURL:     new("c"),
		LicensePhotoURL:    new("d"),
		CirculationCardURL: new("e"),
		VehiclePhoto1URL:   new("1"),
		VehiclePhoto2URL:   new("2"),
		VehiclePhoto3URL:   new("3"),
		VehiclePhoto4URL:   new("4"),
		VehicleType:        new("bici"),
	}
	c := p.Completion()
	assert.True(t, c.IsComplete)
	assert.Equal(t, 10, c.RequiredCount)
	assert.NotContains(t, c.Missing, "vehicle_plate")
}

func TestRiderProfile_Completion_MotoNeedsPlate(t *testing.T) {
	p := &models.RiderProfile{
		ProfilePhotoURL:    new("a"),
		IDPhotoURL:         new("b"),
		CarnetPhotoURL:     new("c"),
		LicensePhotoURL:    new("d"),
		CirculationCardURL: new("e"),
		VehiclePhoto1URL:   new("1"),
		VehiclePhoto2URL:   new("2"),
		VehiclePhoto3URL:   new("3"),
		VehiclePhoto4URL:   new("4"),
		VehicleType:        new("moto"),
		VehiclePlate:       new("AB123CD"),
	}
	assert.True(t, p.IsComplete())
}
