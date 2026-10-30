package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// StaffZoneFareRulesHandler es el CRUD de staff para administrar las zonas
// de tarifa (issue #376): la mínima y tarifa/km por banda horaria que una
// zona geográfica (ej. Charallave) usa en vez de la fórmula global.
type StaffZoneFareRulesHandler interface {
	CreateZoneFareRule(c echo.Context) error
	ListZoneFareRules(c echo.Context) error
	GetZoneFareRule(c echo.Context) error
	UpdateZoneFareRule(c echo.Context) error
	DeactivateZoneFareRule(c echo.Context) error
}

type staffZoneFareRulesHandler struct {
	svc ports.ZoneFareRulesApp
}

func NewStaffZoneFareRulesHandler(svc ports.ZoneFareRulesApp) StaffZoneFareRulesHandler {
	return &staffZoneFareRulesHandler{svc: svc}
}

// CreateZoneFareRule godoc
// @Summary      Crear zona de tarifa
// @Description  Crea una zona con su mínima y tarifa extra/km por banda horaria (día/noche/madrugada) e included_km único para las tres. El polígono (lat/lng, ej. pegado de geojson.io) se convierte a celdas H3 res. 7 en el momento de crear. Los viajes cuyo pickup caiga en una de esas celdas usan esta regla en vez de la fórmula global.
// @Tags         Trips Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.ZoneFareRuleRequest  true  "Datos de la zona"
// @Success      201  {object}  dtos.ZoneFareRuleResponse
// @Failure      400  {string}  string  "payload inválido o polígono inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Router       /trips/staff/zone-fare-rules [post]
func (h *staffZoneFareRulesHandler) CreateZoneFareRule(c echo.Context) error {
	var body dtos.ZoneFareRuleRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	resp, err := h.svc.CreateZoneFareRule(c.Request().Context(), body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusCreated, resp)
}

// ListZoneFareRules godoc
// @Summary      Listar zonas de tarifa (staff)
// @Description  Lista todas las zonas (activas e inactivas), con sus celdas H3.
// @Tags         Trips Staff
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.ZoneFareRulesResponse
// @Failure      401  {string}  string  "No autorizado"
// @Router       /trips/staff/zone-fare-rules [get]
func (h *staffZoneFareRulesHandler) ListZoneFareRules(c echo.Context) error {
	resp, err := h.svc.ListZoneFareRules(c.Request().Context())
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// GetZoneFareRule godoc
// @Summary      Detalle de zona de tarifa (staff)
// @Tags         Trips Staff
// @Produce      json
// @Security     SessionCookie
// @Param        zone_id  path  string  true  "Zone ID"
// @Success      200  {object}  dtos.ZoneFareRuleResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /trips/staff/zone-fare-rules/{zone_id} [get]
func (h *staffZoneFareRulesHandler) GetZoneFareRule(c echo.Context) error {
	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "zone_id inválido")
	}
	resp, err := h.svc.GetZoneFareRule(c.Request().Context(), zoneID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// UpdateZoneFareRule godoc
// @Summary      Reemplazar zona de tarifa
// @Description  Reemplazo completo: precios y polígono (las celdas se recalculan y sustituyen por completo). No es un parche parcial.
// @Tags         Trips Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        zone_id  path  string  true  "Zone ID"
// @Param        body     body  dtos.ZoneFareRuleRequest  true  "Datos de la zona"
// @Success      200  {object}  dtos.ZoneFareRuleResponse
// @Failure      400  {string}  string  "zone_id inválido, payload inválido o polígono inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /trips/staff/zone-fare-rules/{zone_id} [patch]
func (h *staffZoneFareRulesHandler) UpdateZoneFareRule(c echo.Context) error {
	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "zone_id inválido")
	}
	var body dtos.ZoneFareRuleRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	resp, err := h.svc.UpdateZoneFareRule(c.Request().Context(), zoneID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// DeactivateZoneFareRule godoc
// @Summary      Desactivar zona de tarifa
// @Description  Soft delete (active=false): los viajes ya cotizados en esta zona conservan su zone_fare_rule_id. Los pickups en sus celdas vuelven a usar el fallback global.
// @Tags         Trips Staff
// @Produce      json
// @Security     SessionCookie
// @Param        zone_id  path  string  true  "Zone ID"
// @Success      204  "Sin contenido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /trips/staff/zone-fare-rules/{zone_id} [delete]
func (h *staffZoneFareRulesHandler) DeactivateZoneFareRule(c echo.Context) error {
	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "zone_id inválido")
	}
	if err := h.svc.DeactivateZoneFareRule(c.Request().Context(), zoneID); err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
