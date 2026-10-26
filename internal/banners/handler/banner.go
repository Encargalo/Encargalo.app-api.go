package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/banners/domain/dtos"
	"Encargalo.app-api.go/internal/banners/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// BannersHandler es la cara que consume la app (Customer): solo expone los
// banners activos, ya ordenados.
type BannersHandler interface {
	ListActiveBanners(c echo.Context) error
}

type bannersHandler struct {
	svc ports.BannersApp
}

func NewBannersHandler(svc ports.BannersApp) BannersHandler {
	return &bannersHandler{svc: svc}
}

// ListActiveBanners godoc
// @Summary      Listar banners publicitarios activos
// @Description  Devuelve los banners activos ordenados por position, para mostrar en el carrusel del home. Cacheado 1h en Redis.
// @Tags         Banners
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.BannersResponse
// @Failure      401  {string}  string  "No autorizado"
// @Router       /banners [get]
func (h *bannersHandler) ListActiveBanners(c echo.Context) error {
	resp, err := h.svc.ListActiveBanners(c.Request().Context())
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// StaffBannersHandler es el CRUD de staff para administrar los banners.
type StaffBannersHandler interface {
	CreateBanner(c echo.Context) error
	ListBanners(c echo.Context) error
	GetBanner(c echo.Context) error
	UpdateBanner(c echo.Context) error
	DeleteBanner(c echo.Context) error
}

type staffBannersHandler struct {
	svc ports.BannersApp
}

func NewStaffBannersHandler(svc ports.BannersApp) StaffBannersHandler {
	return &staffBannersHandler{svc: svc}
}

// CreateBanner godoc
// @Summary      Crear banner publicitario
// @Description  Crea un banner activo. La imagen va como data-URL/base64 (se sube a Cloudinary en WebP).
// @Tags         Banners Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.CreateBannerRequest  true  "Datos del banner"
// @Success      201  {object}  dtos.BannerResponse
// @Failure      400  {string}  string  "payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      422  {string}  string  "error al subir la imagen"
// @Router       /banners/staff [post]
func (h *staffBannersHandler) CreateBanner(c echo.Context) error {
	var body dtos.CreateBannerRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	resp, err := h.svc.CreateBanner(c.Request().Context(), body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusCreated, resp)
}

// ListBanners godoc
// @Summary      Listar banners (staff)
// @Description  Lista todos los banners (activos e inactivos) ordenados por position.
// @Tags         Banners Staff
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.BannersResponse
// @Failure      401  {string}  string  "No autorizado"
// @Router       /banners/staff [get]
func (h *staffBannersHandler) ListBanners(c echo.Context) error {
	resp, err := h.svc.ListBanners(c.Request().Context())
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// GetBanner godoc
// @Summary      Detalle de banner (staff)
// @Tags         Banners Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Banner ID"
// @Success      200  {object}  dtos.BannerResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /banners/staff/{id} [get]
func (h *staffBannersHandler) GetBanner(c echo.Context) error {
	bannerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	resp, err := h.svc.GetBanner(c.Request().Context(), bannerID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// UpdateBanner godoc
// @Summary      Actualizar banner por partes
// @Description  Actualización parcial: los campos omitidos no se tocan. `image` reemplaza la imagen (se sube a Cloudinary y la anterior se borra, best-effort). `link_url` en `null` remueve el link.
// @Tags         Banners Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path  string  true  "Banner ID"
// @Param        body  body  dtos.UpdateBannerRequest  true  "Campos parciales"
// @Success      200  {object}  dtos.BannerResponse
// @Failure      400  {string}  string  "id inválido / payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Failure      422  {string}  string  "error al subir la imagen"
// @Router       /banners/staff/{id} [patch]
func (h *staffBannersHandler) UpdateBanner(c echo.Context) error {
	bannerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	var body dtos.UpdateBannerRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	resp, err := h.svc.UpdateBanner(c.Request().Context(), bannerID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// DeleteBanner godoc
// @Summary      Eliminar banner
// @Description  Borra el banner y, best-effort, su imagen en Cloudinary.
// @Tags         Banners Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Banner ID"
// @Success      204  "Sin contenido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /banners/staff/{id} [delete]
func (h *staffBannersHandler) DeleteBanner(c echo.Context) error {
	bannerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	if err := h.svc.DeleteBanner(c.Request().Context(), bannerID); err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
