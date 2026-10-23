package googlemaps

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
)

// Defaults de idioma/región; config.Google.LanguageCode/RegionCode los sobreescriben por entorno.
const (
	defaultLanguageCode = "es"
	defaultRegionCode   = "VE"
)

// ErrNoResults indica que Google respondió correctamente pero no encontró
// nada para la consulta (coordenada sin dirección, place_id inexistente) —
// se distingue de un fallo real para que las capas superiores lo mapeen a
// "no encontrado" en vez de a un error genérico.
var ErrNoResults = errors.New("google maps: no results")

// ErrPartialResults indica que la consulta se resolvió solo en parte: el llamador recibe los resultados buenos y decide qué hacer con el hueco.
var ErrPartialResults = errors.New("google maps: partial results")

// httpClient es compartido por todas las llamadas a Google Maps Platform de
// este paquete. El timeout acota cuánto puede colgar una respuesta lenta de
// Google — sin él, una goroutine queda bloqueada indefinidamente por request.
var httpClient = &http.Client{Timeout: 5 * time.Second}

// languageCode es el idioma en que se le piden las direcciones a Google.
func languageCode() string {
	return orDefault(config.Get().Google.LanguageCode, defaultLanguageCode)
}

// regionCode es el único país que atiende el servicio, en formato ccTLD de dos letras.
func regionCode() string {
	return orDefault(config.Get().Google.RegionCode, defaultRegionCode)
}

// matchesRegion filtra por país las respuestas de Google que no aceptan el filtro en la petición
// (reverse geocoding y place details): un country vacío también se descarta, porque sin país no hay
// forma de afirmar que la dirección está dentro del área de servicio.
func matchesRegion(countryCode string) bool {
	return countryCode != "" && strings.EqualFold(countryCode, regionCode())
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
