package handler

import (
	"errors"
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
)

type RiderApplicationHandler interface {
	GetApplication(c echo.Context) error
	SaveApplication(c echo.Context) error
	SubmitApplication(c echo.Context) error
}

type riderApplicationHandler struct {
	svc ports.RiderProfileApp
}

func NewRiderApplicationHandler(svc ports.RiderProfileApp) RiderApplicationHandler {
	return &riderApplicationHandler{svc: svc}
}

// GetApplication godoc
//
// @Summary      Ver la propia solicitud para ser conductor
// @Description  Devuelve la solicitud del customer autenticado: estado, datos del vehículo, fotos subidas y `completion` con lo que falta. Si aún no tiene solicitud, responde una vacía en `draft`.
// @Description  `can_edit` indica si puede guardar y enviar (`draft` o `changes_requested`). `review_reason` trae el motivo cuando el staff rechaza o pide cambios.
// @Description  Si el staff ya aprobó la solicitud, responde 200 con `status: active` (no 409): el 409 `already_rider` queda solo para cuentas rider legacy sin expediente en `riders.profiles`.
// @Tags         Riders Application
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.RiderApplicationResponse
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      409  {string}  string  "Cuenta rider legacy sin expediente (already_rider)"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /riders/application [get]
func (h *riderApplicationHandler) GetApplication(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.GetOwnApplication(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// SaveApplication godoc
//
// @Summary      Guardar la propia solicitud para ser conductor
// @Description  Crea o actualiza parcialmente la solicitud del customer autenticado: los campos omitidos no se tocan. Las fotos van como data-URL/base64 y se suben a Cloudinary.
// @Description  No la envía a revisión, no asigna el rol Rider y no cambia el estado. Solo se puede guardar en `draft` o `changes_requested`.
// @Description  Límites: body de 16MB, cada foto de hasta 7MB en base64 y 30 guardados por hora por cuenta. Una foto reemplazada se borra de Cloudinary.
// @Tags         Riders Application
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.RiderApplicationRequest  true  "Campos parciales"
// @Success      200  {object}  dtos.RiderApplicationResponse
// @Failure      400  {string}  string  "Payload inválido: vehicle_type fuera de moto|bici|carro, texto o foto demasiado largos"
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      409  {string}  string  "La cuenta ya es rider (already_rider) o la solicitud no se puede modificar (rider_application_locked)"
// @Failure      413  {string}  string  "El body supera 16MB"
// @Failure      422  {string}  string  "No se pudo subir una imagen"
// @Failure      429  {string}  string  "Demasiados guardados seguidos"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /riders/application [put]
func (h *riderApplicationHandler) SaveApplication(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var body dtos.RiderApplicationRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.SaveOwnApplication(ctx, session.UserID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// SubmitApplication godoc
//
// @Summary      Enviar a revisión la propia solicitud para ser conductor
// @Description  Pasa la solicitud a `pending_review` para que el staff la revise. Exige que esté completa; si no, responde 422 con `missing`.
// @Description  No asigna el rol Rider: `POST /auth/sign-in/riders` sigue rechazando la cuenta hasta que el staff apruebe.
// @Tags         Riders Application
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.RiderApplicationResponse
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      409  {string}  string  "La cuenta ya es rider (already_rider) o la solicitud no se puede enviar (rider_application_locked)"
// @Failure      422  {object}  dtos.ActivateErrorResponse  "Solicitud incompleta"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /riders/application/submit [post]
func (h *riderApplicationHandler) SubmitApplication(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.SubmitOwnApplication(ctx, session.UserID)
	if errors.Is(err, errcustom.ErrRiderProfileIncomplete) {
		missing := resp.Completion.Missing
		if missing == nil {
			missing = []string{}
		}
		return response.SendError(c, http.StatusUnprocessableEntity,
			dtos.ActivateErrorResponse{
				Message: errcustom.ErrRiderProfileIncomplete.Error(),
				Missing: missing,
			},
			errcustom.Code(errcustom.ErrRiderProfileIncomplete),
			errcustom.ErrRiderProfileIncomplete.Error(),
			map[string]any{"missing": missing},
		)
	}
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}
