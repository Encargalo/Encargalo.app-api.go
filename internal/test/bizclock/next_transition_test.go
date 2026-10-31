package bizclock_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextTransition_NoWindows(t *testing.T) {
	opensAt, closesAt := bizclock.NextTransition(nil, at(9, 0))
	assert.Nil(t, opensAt)
	assert.Nil(t, closesAt)
}

func TestNextTransition_CurrentlyOpen(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: true},
	}

	opensAt, closesAt := bizclock.NextTransition(windows, at(9, 30))
	assert.Nil(t, opensAt)
	require.NotNil(t, closesAt)
	assert.Equal(t, at(11, 0), *closesAt)
}

func TestNextTransition_ClosedOpensLaterToday(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "18:00:00", EndTime: "23:00:00", Enabled: true},
	}

	opensAt, closesAt := bizclock.NextTransition(windows, at(17, 45))
	assert.Nil(t, closesAt)
	require.NotNil(t, opensAt)
	assert.Equal(t, at(18, 0), *opensAt)
}

func TestNextTransition_ClosedOpensTomorrow(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: true},
	}

	opensAt, closesAt := bizclock.NextTransition(windows, at(12, 0))
	assert.Nil(t, closesAt)
	require.NotNil(t, opensAt)
	assert.Equal(t, at(6, 0).AddDate(0, 0, 1), *opensAt)
}

func TestNextTransition_CrossesMidnight_CurrentlyOpen(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "18:00:00", EndTime: "02:00:00", Enabled: true},
	}

	opensAt, closesAt := bizclock.NextTransition(windows, at(23, 0))
	assert.Nil(t, opensAt)
	require.NotNil(t, closesAt)
	assert.Equal(t, at(2, 0).AddDate(0, 0, 1), *closesAt)
}

func TestNextTransition_DisabledWindowIgnored(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: false},
	}

	opensAt, closesAt := bizclock.NextTransition(windows, at(9, 0))
	assert.Nil(t, opensAt)
	assert.Nil(t, closesAt)
}

func TestNextTransition_DayOfWeekRestriction(t *testing.T) {
	// at(2026,1,5) es lunes (weekday 1); el martes (2) siguiente es 2026-01-06.
	tuesday := int16(2)
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: true, DayOfWeek: &tuesday},
	}

	opensAt, closesAt := bizclock.NextTransition(windows, at(9, 0))
	assert.Nil(t, closesAt)
	require.NotNil(t, opensAt)
	assert.Equal(t, time.Date(2026, time.January, 6, 6, 0, 0, 0, time.UTC), *opensAt)
}
