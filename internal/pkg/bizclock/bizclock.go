package bizclock

import (
	"log/slog"
	"time"
)

const zoneName = "America/Caracas"

var location = loadLocation()

func loadLocation() *time.Location {
	loc, err := time.LoadLocation(zoneName)
	if err != nil {
		slog.Error("no se pudo cargar zona horaria de negocio, se usa UTC", "zone", zoneName, "error", err)
		return time.UTC
	}
	return loc
}

// Now retorna la hora actual en la zona horaria de negocio (America/Caracas).
func Now() time.Time {
	return time.Now().In(location)
}

// Window representa una ventana de disponibilidad (una fila de products.schedules)
// en su forma primitiva, desacoplada de bun/dtos para poder usarse desde cualquier dominio.
type Window struct {
	DayOfWeek *int16
	StartTime string
	EndTime   string
	Enabled   bool
}

// IsAvailableAt evalúa si `now` cae dentro de alguna ventana habilitada.
// Sin ventanas configuradas, siempre está disponible.
func IsAvailableAt(windows []Window, now time.Time) bool {
	if len(windows) == 0 {
		return true
	}

	minute := now.Hour()*60 + now.Minute()
	weekday := int(now.Weekday())

	for _, w := range windows {
		if !w.Enabled {
			continue
		}
		if !appliesToDay(w.DayOfWeek, weekday) {
			continue
		}
		if coversMinute(w.StartTime, w.EndTime, minute) {
			return true
		}
	}

	return false
}

func appliesToDay(dayOfWeek *int16, weekday int) bool {
	if dayOfWeek == nil {
		return true
	}
	return int(*dayOfWeek) == weekday
}

func coversMinute(startTime, endTime string, minute int) bool {
	start, ok := minutesSinceMidnight(startTime)
	if !ok {
		return false
	}
	end, ok := minutesSinceMidnight(endTime)
	if !ok {
		return false
	}
	if start <= end {
		return minute >= start && minute < end
	}
	// Rango que cruza medianoche (ej. 18:00 -> 02:00).
	return minute >= start || minute < end
}

func minutesSinceMidnight(hhmmss string) (int, bool) {
	t, err := time.Parse("15:04:05", hhmmss)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}
