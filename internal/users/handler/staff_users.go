package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	authPorts "Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffUsersHandler interface {
	ListUsers(c echo.Context) error
	FindCandidate(c echo.Context) error
	GetUser(c echo.Context) error
	UpdateUser(c echo.Context) error
	BlockUser(c echo.Context) error
	UnblockUser(c echo.Context) error
}

type staffUsersHandler struct {
	svc  ports.StaffUsersApp
	auth authPorts.AuthApp
}

func NewStaffUsersHandler(svc ports.StaffUsersApp, auth authPorts.AuthApp) StaffUsersHandler {
	return &staffUsersHandler{svc: svc, auth: auth}
}

// ListUsers godoc
// @Summary      Listar clientes gestionables por staff
// @Description  Lista todas las cuentas no eliminadas con paginación por cursor (created_at DESC, id DESC). El cursor solo es válido con los mismos valores de q, status y limit.
// @Tags         Users Staff
// @Produce      json
// @Security     SessionCookie
// @Param        cursor  query  string  false  "Cursor opaco de la página anterior"
// @Param        limit   query  int     false  "Tamaño de página (default 20, max 100)"
// @Param        q       query  string  false  "Búsqueda por nombre, apellido, email o teléfono (mín. 2 caracteres)"
// @Param        status  query  string  false  "all|active|blocked"  default(all)
// @Success      200  {object}  dto.StaffUserListResponse
// @Failure      400  {string}  string
// @Failure      401  {string}  string
// @Router       /users/staff [get]
func (h *staffUsersHandler) ListUsers(c echo.Context) error {
	limit := 0
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "limit inválido")
		}
		limit = n
	}

	resp, err := h.svc.ListUsers(c.Request().Context(), dto.ListStaffUsersQuery{
		Cursor: c.QueryParam("cursor"),
		Limit:  limit,
		Query:  c.QueryParam("q"),
		Status: c.QueryParam("status"),
	})
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// FindCandidate godoc
// @Summary      Buscar cliente por teléfono
// @Description  Busca cualquier cuenta no eliminada por teléfono exacto.
// @Tags         Users Staff
// @Produce      json
// @Security     SessionCookie
// @Param        phone  query  string  true  "Teléfono E.164"
// @Success      200  {object}  dto.StaffUserCandidateResponse
// @Failure      400  {string}  string
// @Failure      404  {string}  string
// @Router       /users/staff/candidates [get]
func (h *staffUsersHandler) FindCandidate(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	phone := strings.TrimSpace(c.QueryParam("phone"))
	if phone == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "phone requerido")
	}
	slog.Info("staff buscó cliente por teléfono", "staffID", session.UserID, "phone", phone)

	resp, err := h.svc.FindCandidate(ctx, phone)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// GetUser godoc
// @Summary      Detalle de cliente
// @Description  Retorna el detalle de cualquier cuenta no eliminada.
// @Tags         Users Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "ID del usuario"
// @Success      200  {object}  dto.StaffUserDetailResponse
// @Failure      404  {string}  string
// @Router       /users/staff/{userId} [get]
func (h *staffUsersHandler) GetUser(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}
	resp, err := h.svc.GetUser(c.Request().Context(), userID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// UpdateUser godoc
// @Summary      Actualizar cliente
// @Description  Actualización parcial de name, sur_name y email. El teléfono no es editable por staff.
// @Tags         Users Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "ID del usuario"
// @Param        body    body  dto.StaffUpdateUserRequest  true  "Campos a actualizar"
// @Success      200  {object}  dto.StaffUserDetailResponse
// @Failure      400  {string}  string
// @Failure      404  {string}  string
// @Failure      409  {string}  string  "Email duplicado"
// @Router       /users/staff/{userId} [patch]
func (h *staffUsersHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}

	var body dto.StaffUpdateUserRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if body.Name == nil && body.SurName == nil && body.Email == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "al menos un campo es requerido")
	}

	resp, err := h.svc.UpdateUser(ctx, session.UserID, userID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// BlockUser godoc
// @Summary      Bloquear cliente
// @Description  Deshabilita la cuenta (enabled=false) e invalida todas sus sesiones activas.
// @Tags         Users Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "ID del usuario"
// @Success      200  {object}  dto.StaffUserDetailResponse
// @Failure      404  {string}  string
// @Router       /users/staff/{userId}/block [post]
func (h *staffUsersHandler) BlockUser(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}

	resp, err := h.svc.BlockUser(ctx, session.UserID, userID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	if err := h.auth.DeleteAllSessionsForUser(ctx, userID); err != nil {
		applog.Error(ctx, "staff bloqueó usuario pero no pudo invalidar sesiones", err, "staff_id", session.UserID.String(), "user_id", userID.String())
	}

	return c.JSON(http.StatusOK, resp)
}

// UnblockUser godoc
// @Summary      Desbloquear cliente
// @Description  Rehabilita la cuenta (enabled=true).
// @Tags         Users Staff
// @Produce      json
// @Security     SessionCookie
// @Param        userId  path  string  true  "ID del usuario"
// @Success      200  {object}  dto.StaffUserDetailResponse
// @Failure      404  {string}  string
// @Router       /users/staff/{userId}/unblock [post]
func (h *staffUsersHandler) UnblockUser(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "userId inválido")
	}

	resp, err := h.svc.UnblockUser(ctx, session.UserID, userID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}
