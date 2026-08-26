package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffShopHandler interface {
	FindOwnerCandidate(c echo.Context) error
	CreateDraftShop(c echo.Context) error
	ListShops(c echo.Context) error
	GetShop(c echo.Context) error
	UpdateDraftShop(c echo.Context) error
	ActivateShop(c echo.Context) error
}

type staffShopHandler struct {
	svc ports.StaffShopApp
}

func NewStaffShopHandler(svc ports.StaffShopApp) StaffShopHandler {
	return &staffShopHandler{svc: svc}
}

// FindOwnerCandidate godoc
// @Summary      Buscar candidato a owner por teléfono
// @Description  Busca una cuenta existente por teléfono para vincularla como Owner de un shop nuevo. Sujeto a rate limit por IP.
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        phone  query  string  true  "Teléfono"
// @Success      200  {object}  dtos.OwnerCandidateResponse
// @Failure      400  {string}  string  "phone requerido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string
// @Router       /shops/staff/candidates [get]
func (h *staffShopHandler) FindOwnerCandidate(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	phone := strings.TrimSpace(c.QueryParam("phone"))
	if phone == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "phone requerido")
	}
	// Auditoría: este endpoint expone PII (nombre, email, teléfono) de
	// cualquier cuenta por número de teléfono; deja rastro de qué staff
	// consultó qué número.
	slog.Info("staff buscó candidato a owner por teléfono", "staffID", session.UserID, "phone", phone)
	resp, err := h.svc.FindOwnerCandidate(ctx, phone)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// CreateDraftShop godoc
// @Summary      Crear shop en borrador
// @Description  Crea el shop en estado draft y vincula el rol Owner (sin remover los roles que la cuenta ya tenga) a la cuenta indicada.
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.CreateDraftShopRequest  true  "Datos mínimos"
// @Success      201  {object}  dtos.StaffShopResponse
// @Failure      400  {string}  string
// @Failure      401  {string}  string  "No autorizado"
// @Router       /shops/staff [post]
func (h *staffShopHandler) CreateDraftShop(c echo.Context) error {
	var body dtos.CreateDraftShopRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	resp, err := h.svc.CreateDraftShop(c.Request().Context(), body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusCreated, resp)
}

// ListShops godoc
// @Summary      Listar shops en onboarding
// @Description  Lista shops por status (default draft), paginado.
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        status  query  string  false  "draft|active"  default(draft)
// @Param        page    query  int     false  "Página (default 1)"
// @Param        limit   query  int     false  "Límite por página, máx 50 (default 20)"
// @Success      200  {object}  dtos.StaffShopsResponse
// @Failure      400  {string}  string  "status inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Router       /shops/staff [get]
func (h *staffShopHandler) ListShops(c echo.Context) error {
	status := shopsmodels.ShopStatus(strings.TrimSpace(c.QueryParam("status")))
	switch status {
	case "", shopsmodels.ShopStatusDraft, shopsmodels.ShopStatusActive:
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "status inválido")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}

	list, err := h.svc.ListShops(c.Request().Context(), status, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, list)
}

// GetShop godoc
// @Summary      Detalle de shop (staff)
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Shop ID"
// @Success      200  {object}  dtos.StaffShopResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /shops/staff/{id} [get]
func (h *staffShopHandler) GetShop(c echo.Context) error {
	shopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	resp, err := h.svc.GetShop(c.Request().Context(), shopID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// UpdateDraftShop godoc
// @Summary      Completar shop por partes
// @Description  Actualización parcial: los campos omitidos no se tocan; enviar un campo en `null` lo limpia. logo_image/banner van como data-URL/base64 (se suben a Cloudinary). Rechaza la edición mientras el local esté abierto (`opened: true`).
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path  string  true  "Shop ID"
// @Param        body  body  dtos.UpdateDraftShopRequest  true  "Campos parciales"
// @Success      200  {object}  dtos.StaffShopResponse
// @Failure      400  {string}  string  "id inválido / payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      409  {string}  string  "El local está abierto"
// @Router       /shops/staff/{id} [patch]
func (h *staffShopHandler) UpdateDraftShop(c echo.Context) error {
	shopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	var body dtos.UpdateDraftShopRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	resp, err := h.svc.UpdateDraftShop(c.Request().Context(), shopID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// ActivateShop godoc
// @Summary      Activar shop
// @Description  Requiere checklist completo (logo, banner, dirección, teléfono, coordenadas). A partir de aquí el owner puede abrir el local.
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Shop ID"
// @Success      200  {object}  dtos.StaffShopResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      422  {object}  dtos.ShopActivateErrorResponse
// @Router       /shops/staff/{id}/activate [post]
func (h *staffShopHandler) ActivateShop(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	shopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	resp, err := h.svc.ActivateShop(ctx, shopID, session.UserID)
	if err != nil {
		if errors.Is(err, errcustom.ErrShopOnboardingIncomplete) {
			shop, getErr := h.svc.GetShop(ctx, shopID)
			missing := []string{}
			if getErr == nil {
				missing = shop.Completion.Missing
			}
			return response.SendError(c, http.StatusUnprocessableEntity,
				dtos.ShopActivateErrorResponse{
					Message: errcustom.ErrShopOnboardingIncomplete.Error(),
					Missing: missing,
				},
				errcustom.Code(errcustom.ErrShopOnboardingIncomplete),
				errcustom.ErrShopOnboardingIncomplete.Error(),
				map[string]any{"missing": missing},
			)
		}
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}
