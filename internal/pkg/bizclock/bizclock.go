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
		// Fallback a UTC-4 fijo (Venezuela no usa horario de verano, así que
		// esto es exactamente correcto, no solo una aproximación). Caer a
		// UTC en vez de esto desfasaría los horarios de los locales 4h.
		// Esto solo debería ocurrir si el binario no se compiló con
		// `-tags timetzdata` (ver Dockerfile) en una imagen sin
		// /usr/share/zoneinfo.
		slog.Error("no se pudo cargar zona horaria de negocio, se usa UTC-4 fijo", "zone", zoneName, "error", err)
		return time.FixedZone("VET", -4*60*60)
	}
	return loc
}

// Now retorna la hora actual en la zona horaria de negocio (America/Caracas).
func Now() time.Time {
	return time.Now().In(location)
}

// Location es la zona horaria de negocio. Sirve para reinterpretar timestamps
// ya persistidos en UTC, como hace el ETL al calcular hora del día.
func Location() *time.Location {
	return location
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

// NextTransition calcula, a partir de `now`, el cierre de la ventana que lo
// cubre en este instante (closesAt) o, si ninguna lo cubre, el inicio de la
// próxima ventana habilitada (opensAt). Sin ventanas habilitadas retorna
// (nil, nil): horario desconocido.
func NextTransition(windows []Window, now time.Time) (opensAt, closesAt *time.Time) {
	type interval struct {
		start, end time.Time
	}

	var intervals []interval
	for _, w := range windows {
		if !w.Enabled {
			continue
		}
		startMin, ok := minutesSinceMidnight(w.StartTime)
		if !ok {
			continue
		}
		endMin, ok := minutesSinceMidnight(w.EndTime)
		if !ok {
			continue
		}

		for offset := -1; offset <= 7; offset++ {
			day := now.AddDate(0, 0, offset)
			if !appliesToDay(w.DayOfWeek, int(day.Weekday())) {
				continue
			}
			start := time.Date(day.Year(), day.Month(), day.Day(), startMin/60, startMin%60, 0, 0, now.Location())
			end := time.Date(day.Year(), day.Month(), day.Day(), endMin/60, endMin%60, 0, 0, now.Location())
			if !end.After(start) {
				end = end.AddDate(0, 0, 1)
			}
			intervals = append(intervals, interval{start, end})
		}
	}

	for _, iv := range intervals {
		if !now.Before(iv.start) && now.Before(iv.end) {
			end := iv.end
			return nil, &end
		}
	}

	var next *time.Time
	for _, iv := range intervals {
		if iv.start.After(now) && (next == nil || iv.start.Before(*next)) {
			start := iv.start
			next = &start
		}
	}
	return next, nil
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
