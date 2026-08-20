package handler

import (
	"net/http"
	"strconv"

	"Encargalo.app-api.go/internal/maps/domain/models"
	"Encargalo.app-api.go/internal/maps/domain/ports"
	"github.com/labstack/echo/v4"
)

type MapsHandler interface {
	ReverseGeocode(e echo.Context) error
	AutocompletePlaces(e echo.Context) error
	GetPlaceDetails(e echo.Context) error
}

type mapsHandler struct {
	svc ports.MapsApp
}

func NewMapsHandler(svc ports.MapsApp) MapsHandler {
	return &mapsHandler{svc}
}

// ReverseGeocode godoc
// @Summary      Geocoding inverso
// @Description  Resuelve una dirección a partir de una coordenada, usando la Google Geocoding API. Incluye formatted_address, parts (locality=ciudad, administrative_area_level_1=estado, administrative_area_level_2=municipio/provincia) y address_components crudos de Google para que el cliente arme la dirección. Cacheado por el servidor por coordenadas redondeadas.
// @Tags         Maps
// @Produce      json
// @Security     SessionCookie
// @Param        lat  query     number  true  "Latitud"
// @Param        lng  query     number  true  "Longitud"
// @Success      200  {object}  models.ReverseGeocodeResult
// @Failure      400  {string}  string  "lat/lng requeridos o fuera de rango"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Sin resultados para esa coordenada"
// @Failure      429  {string}  string  "Demasiadas solicitudes"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /maps/geocode/reverse [get]
func (h *mapsHandler) ReverseGeocode(e echo.Context) error {
	ctx := e.Request().Context()

	lat, err := strconv.ParseFloat(e.QueryParam("lat"), 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "lat es requerido y debe ser numérico")
	}
	lng, err := strconv.ParseFloat(e.QueryParam("lng"), 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "lng es requerido y debe ser numérico")
	}

	var result *models.ReverseGeocodeResult
	result, err = h.svc.ReverseGeocode(ctx, lat, lng)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, result)
}

// Autocomplete godoc
// @Summary      Autocompletar lugares
// @Description  Retorna sugerencias de lugares para un texto de búsqueda, usando la Google Places Autocomplete API. session_token debe ser el mismo UUID usado luego en /maps/places/details para agrupar la sesión de facturación en Google. query debe tener al menos 3 caracteres.
// @Tags         Maps
// @Produce      json
// @Security     SessionCookie
// @Param        query          query     string  true   "Texto de búsqueda (mínimo 3 caracteres)"
// @Param        lat            query     number  false  "Latitud para sesgar resultados cercanos"
// @Param        lng            query     number  false  "Longitud para sesgar resultados cercanos"
// @Param        session_token  query     string  true   "UUID de sesión de búsqueda"
// @Success      200            {array}   models.PlacePrediction
// @Failure      400            {string}  string  "query muy corto, session_token requerido o coordenadas de sesgo fuera de rango"
// @Failure      401            {string}  string  "No autorizado"
// @Failure      429            {string}  string  "Demasiadas solicitudes"
// @Failure      500            {string}  string  "Error interno del servidor"
// @Router       /maps/places/autocomplete [get]
func (h *mapsHandler) AutocompletePlaces(e echo.Context) error {
	ctx := e.Request().Context()

	query := e.QueryParam("query")

	sessionToken := e.QueryParam("session_token")
	if sessionToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "session_token es requerido")
	}

	biasLat, biasLng := parseOptionalCoords(e)

	var predictions []models.PlacePrediction
	predictions, err := h.svc.AutocompletePlaces(ctx, query, biasLat, biasLng, sessionToken)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, predictions)
}

// PlaceDetails godoc
// @Summary      Detalle de un lugar
// @Description  Resuelve dirección y coordenadas de un place_id devuelto por /maps/places/autocomplete, usando la Google Places Details API. Incluye address, lat/lng, parts (locality=ciudad, administrative_area_level_1=estado, administrative_area_level_2=municipio/provincia) y address_components crudos de Google para que el cliente arme la dirección. Cacheado por el servidor por place_id.
// @Tags         Maps
// @Produce      json
// @Security     SessionCookie
// @Param        place_id       query     string  true  "ID de lugar de Google"
// @Param        session_token  query     string  true  "UUID de sesión de búsqueda (el mismo usado en autocomplete)"
// @Success      200            {object}  models.PlaceDetails
// @Failure      400            {string}  string  "place_id/session_token requeridos"
// @Failure      401            {string}  string  "No autorizado"
// @Failure      429            {string}  string  "Demasiadas solicitudes"
// @Failure      500            {string}  string  "Error interno del servidor"
// @Router       /maps/places/details [get]
func (h *mapsHandler) GetPlaceDetails(e echo.Context) error {
	ctx := e.Request().Context()

	placeID := e.QueryParam("place_id")
	if placeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "place_id es requerido")
	}

	sessionToken := e.QueryParam("session_token")
	if sessionToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "session_token es requerido")
	}

	var details *models.PlaceDetails
	details, err := h.svc.GetPlaceDetails(ctx, placeID, sessionToken)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, details)
}

// parseOptionalCoords reads lat/lng as an optional bias pair — absent or
// unparsable values simply mean "no bias", not an error.
func parseOptionalCoords(e echo.Context) (*float64, *float64) {
	lat, latErr := strconv.ParseFloat(e.QueryParam("lat"), 64)
	lng, lngErr := strconv.ParseFloat(e.QueryParam("lng"), 64)
	if latErr != nil || lngErr != nil {
		return nil, nil
	}
	return &lat, &lng
}
