package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	ports "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/labstack/echo/v4"
)

type Shops interface {
	GetAllShops(c echo.Context) error
	GetShopsBy(c echo.Context) error
	GetMyShop(c echo.Context) error
}

type shops struct {
	app ports.ShopsApp
}

func NewShopsHandler(app ports.ShopsApp) Shops {
	return &shops{app}
}

// GetMyShop godoc
// @Summary      Obtener información del negocio propio
// @Description  Retorna la información del negocio asociado a la sesión del empleado autenticado.
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.ShopResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Negocio no encontrado"
// @Failure      422  {string}  string  "El usuario no tiene un negocio asociado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/shops/me [get]
func (s *shops) GetMyShop(c echo.Context) error {
	ctx := c.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	shop, err := s.app.GetShopByID(ctx, *session.ShopID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, shop)
}

// GetAllShops godoc
// @Summary      Obtener todas las tiendas cercanas
// @Description  Retorna todas las tiendas disponibles ordenadas por proximidad a las coordenadas proporcionadas.
// @Tags         Shops
// @Produce      json
// @Security     SessionCookie
// @Param        lat  query     number  true  "Latitud del usuario, ej: 4.678034"
// @Param        lon  query     number  true  "Longitud del usuario, ej: -74.0496399"
// @Success      200  {array}   dtos.ShopResponse  "Lista de tiendas"
// @Failure      400  {string}  string             "Coordenadas inválidas o faltantes"
// @Failure      401  {string}  string             "No autorizado"
// @Failure      404  {string}  string             "No se encontraron tiendas"
// @Failure      500  {string}  string             "Error interno del servidor"
// @Router       /shops/all [get]
func (s *shops) GetAllShops(c echo.Context) error {

	ctx := c.Request().Context()

	coords := dtos.Coords{}

	if err := c.Bind(&coords); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := coords.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	shops, err := s.app.GetAllShops(ctx, coords)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, shops)
}

// GetShopsBy godoc
// @Summary      Obtener tienda por ID o tag
// @Description  Retorna la información de una tienda junto con sus productos. Se debe enviar al menos uno de los parámetros: `id` o `tag`.
// @Tags         Shops
// @Produce      json
// @Security     SessionCookie
// @Param        id   query     string  false  "ID de la tienda (UUID)"
// @Param        tag  query     string  false  "Tag de la tienda, ej: dmo"
// @Param        lat  query     number  true   "Latitud del usuario, ej: 4.678034"
// @Param        lon  query     number  true   "Longitud del usuario, ej: -74.0496399"
// @Success      200  {array}   dtos.ShopResponse  "Datos de la tienda"
// @Failure      400  {string}  string             "Parámetros inválidos o faltantes"
// @Failure      401  {string}  string             "No autorizado"
// @Failure      404  {string}  string             "Tienda no encontrada"
// @Failure      500  {string}  string             "Error interno del servidor"
// @Router       /shops [get]
func (p *shops) GetShopsBy(c echo.Context) error {

	ctx := c.Request().Context()

	criteria := dtos.SearchShopsByID{}
	coords := dtos.Coords{}

	if err := c.Bind(&criteria); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := criteria.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Bind(&coords); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := coords.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	products, err := p.app.GetShopsBy(ctx, criteria, coords)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, products)
}
