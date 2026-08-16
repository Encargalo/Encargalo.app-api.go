package googlemaps

import (
	"errors"
	"net/http"
	"time"
)

// ErrNoResults indica que Google respondió correctamente pero no encontró
// nada para la consulta (coordenada sin dirección, place_id inexistente) —
// se distingue de un fallo real para que las capas superiores lo mapeen a
// "no encontrado" en vez de a un error genérico.
var ErrNoResults = errors.New("google maps: no results")

// httpClient es compartido por todas las llamadas a Google Maps Platform de
// este paquete. El timeout acota cuánto puede colgar una respuesta lenta de
// Google — sin él, una goroutine queda bloqueada indefinidamente por request.
var httpClient = &http.Client{Timeout: 5 * time.Second}
