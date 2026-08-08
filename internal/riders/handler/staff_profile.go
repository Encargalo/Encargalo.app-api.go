package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffRiderHandler interface {
	FindCandidate(c echo.Context) error
	ListProfiles(c echo.Context) error
	GetProfile(c echo.Context) error
	UpsertProfile(c echo.Context) error
	DeletePhoto(c echo.Context) error
	Activate(c echo.Context) error
	Suspend(c echo.Context) error
	TouchContact(c echo.Context) error
}

type staffRiderHandler struct {
	svc ports.RiderProfileApp
}

func NewStaffRiderHandler(svc ports.RiderProfileApp) StaffRiderHandler {
	return &staffRiderHandler{svc: svc}
}

// FindCandidate godoc
// @Summary      Buscar candidato a rider por teléfono
// @Description  Busca una cuenta (Customer) por teléfono y, si existe, el expediente rider.
// @Tags         Riders Staff
// @Produce      json
// @Security     SessionCookie
// @Param        phone  query  string  true  "Teléfono"
// @Success      200  {object}  dtos.CandidateResponse
// @Failure      404  {string}  string
// @Router       /riders/staff/candidates [get]
func (h *staffRiderHandler) FindCandidate(c echo.Context) error {
	phone := strings.TrimSpace(c.QueryParam("phone"))
	if phone == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "phone requerido")
	}
	resp, err := h.svc.FindCandidate(c.Request().Context(), phone)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ListProfiles godoc
// @Summary      Listar expedientes rider
// @Description  Lista perfiles por status (default draft). Incluye completion para follow-up.
// @Tags         Riders Staff
// @Produce      json
// @Security     SessionCookie
// @Param        status  query  string  false  "draft|active|suspended"  default(draft)
// @Success      200  {array}  dtos.RiderProfileResponse
// @Router       /riders/staff/profiles [get]
func (h *staffRiderHandler) ListProfiles(c echo.Context) error {
	status := models.RiderProfileStatus(strings.TrimSpace(c.QueryParam("status")))
	switch status {
	case "", models.RiderProfileDraft, models.RiderProfileActive, models.RiderProfileSuspended:
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "status inválido")
	}
	list, err := h.svc.ListProfiles(c.Request().Context(), status)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, list)
}

// GetProfile godoc
// @Summary      Detalle expediente rider
// @Tags         Riders Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "User ID"
// @Success      200  {object}  dtos.RiderProfileResponse
// @Router       /riders/staff/profiles/{userId} [get]
func (h *staffRiderHandler) GetProfile(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	resp, err := h.svc.GetProfile(c.Request().Context(), userID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// UpsertProfile godoc
// @Summary      Guardar borrador expediente rider
// @Description  Upsert parcial. Fotos como data-URL/base64 (se suben a Cloudinary). No asigna rol Rider.
// @Tags         Riders Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "User ID"
// @Param        body    body  dtos.UpsertRiderProfileRequest  true  "Campos parciales"
// @Success      200  {object}  dtos.RiderProfileResponse
// @Router       /riders/staff/profiles/{userId} [put]
func (h *staffRiderHandler) UpsertProfile(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	var body dtos.UpsertRiderProfileRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	resp, err := h.svc.UpsertProfile(c.Request().Context(), userID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// DeletePhoto godoc
// @Summary      Eliminar foto del expediente rider
// @Description  Pone NULL el campo en DB y elimina el asset en Cloudinary. field: profile_photo|id_photo|carnet_photo|license_photo|circulation_card|vehicle_photo_1..4
// @Tags         Riders Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "User ID"
// @Param        field   path  string  true  "Campo de foto"
// @Success      200  {object}  dtos.RiderProfileResponse
// @Failure      400  {string}  string
// @Failure      404  {string}  string
// @Router       /riders/staff/profiles/{userId}/photos/{field} [delete]
func (h *staffRiderHandler) DeletePhoto(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	field := strings.TrimSpace(c.Param("field"))
	if !models.ValidRiderPhotoField(field) {
		return echo.NewHTTPError(http.StatusBadRequest, "field inválido")
	}
	resp, err := h.svc.DeletePhoto(c.Request().Context(), userID, field)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Activate godoc
// @Summary      Activar rider
// @Description  Requiere checklist completo. Asigna rol Rider y status=active.
// @Tags         Riders Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "User ID"
// @Success      200  {object}  dtos.RiderProfileResponse
// @Failure      422  {object}  dtos.ActivateErrorResponse
// @Router       /riders/staff/profiles/{userId}/activate [post]
func (h *staffRiderHandler) Activate(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	resp, err := h.svc.Activate(ctx, userID, session.UserID)
	if err != nil {
		if errors.Is(err, errcustom.ErrRiderProfileIncomplete) {
			profile, getErr := h.svc.GetProfile(ctx, userID)
			missing := []string{}
			if getErr == nil {
				missing = profile.Completion.Missing
			}
			return c.JSON(http.StatusUnprocessableEntity, dtos.ActivateErrorResponse{
				Message: errcustom.ErrRiderProfileIncomplete.Error(),
				Missing: missing,
			})
		}
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Suspend godoc
// @Summary      Suspender rider
// @Tags         Riders Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "User ID"
// @Success      200  {object}  dtos.RiderProfileResponse
// @Router       /riders/staff/profiles/{userId}/suspend [post]
func (h *staffRiderHandler) Suspend(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	resp, err := h.svc.Suspend(c.Request().Context(), userID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// TouchContact godoc
// @Summary      Marcar contacto de seguimiento
// @Tags         Riders Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "User ID"
// @Param        body    body  dtos.TouchContactRequest  false  "Notas opcionales"
// @Success      200  {object}  dtos.RiderProfileResponse
// @Router       /riders/staff/profiles/{userId}/touch-contact [post]
func (h *staffRiderHandler) TouchContact(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	var body dtos.TouchContactRequest
	_ = json.NewDecoder(c.Request().Body).Decode(&body)
	resp, err := h.svc.TouchContact(c.Request().Context(), userID, body.Notes)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}
