package models_test

import (
	"testing"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestRiderProfile_ApplicantCanEdit(t *testing.T) {
	cases := map[models.RiderProfileStatus]bool{
		models.RiderProfileDraft:            true,
		models.RiderProfileChangesRequested: true,
		models.RiderProfilePendingReview:    false,
		models.RiderProfileRejected:         false,
		models.RiderProfileActive:           false,
		models.RiderProfileSuspended:        false,
	}
	for status, want := range cases {
		t.Run(string(status), func(t *testing.T) {
			p := &models.RiderProfile{Status: status}
			assert.Equal(t, want, p.ApplicantCanEdit())
		})
	}
}

func TestValidRiderProfileStatus(t *testing.T) {
	for _, status := range []models.RiderProfileStatus{
		models.RiderProfileDraft,
		models.RiderProfilePendingReview,
		models.RiderProfileChangesRequested,
		models.RiderProfileRejected,
		models.RiderProfileActive,
		models.RiderProfileSuspended,
	} {
		assert.True(t, models.ValidRiderProfileStatus(status), status)
	}
	assert.False(t, models.ValidRiderProfileStatus(""))
	assert.False(t, models.ValidRiderProfileStatus("approved"))
}
