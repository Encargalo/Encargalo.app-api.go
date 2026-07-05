package bizclock_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"github.com/stretchr/testify/assert"
)

func at(hour, minute int) time.Time {
	return time.Date(2026, time.January, 5, hour, minute, 0, 0, time.UTC)
}

func TestIsAvailableAt_NoWindows(t *testing.T) {
	assert.True(t, bizclock.IsAvailableAt(nil, at(9, 0)))
	assert.True(t, bizclock.IsAvailableAt([]bizclock.Window{}, at(9, 0)))
}

func TestIsAvailableAt_SimpleRange(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: true},
	}

	assert.True(t, bizclock.IsAvailableAt(windows, at(9, 30)))
	assert.False(t, bizclock.IsAvailableAt(windows, at(12, 0)))
	assert.False(t, bizclock.IsAvailableAt(windows, at(5, 59)))
}

func TestIsAvailableAt_CrossesMidnight(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "18:00:00", EndTime: "02:00:00", Enabled: true},
	}

	assert.True(t, bizclock.IsAvailableAt(windows, at(23, 0)))
	assert.True(t, bizclock.IsAvailableAt(windows, at(1, 0)))
	assert.False(t, bizclock.IsAvailableAt(windows, at(10, 0)))
}

func TestIsAvailableAt_Disabled(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: false},
	}

	assert.False(t, bizclock.IsAvailableAt(windows, at(9, 0)))
}

func TestIsAvailableAt_DayOfWeekMismatch(t *testing.T) {
	// at(2026,1,5) es lunes (weekday 1)
	tuesday := int16(2)
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: true, DayOfWeek: &tuesday},
	}

	assert.False(t, bizclock.IsAvailableAt(windows, at(9, 0)))
}

func TestIsAvailableAt_DayOfWeekMatch(t *testing.T) {
	monday := int16(1)
	windows := []bizclock.Window{
		{StartTime: "06:00:00", EndTime: "11:00:00", Enabled: true, DayOfWeek: &monday},
	}

	assert.True(t, bizclock.IsAvailableAt(windows, at(9, 0)))
}

func TestIsAvailableAt_InvalidTimeFormat(t *testing.T) {
	windows := []bizclock.Window{
		{StartTime: "6pm", EndTime: "02:00:00", Enabled: true},
	}

	assert.NotPanics(t, func() {
		assert.False(t, bizclock.IsAvailableAt(windows, at(19, 0)))
	})
}
