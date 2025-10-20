package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	ports "Encargalo.app-api.go/internal/shops/domain/ports/shops"
	"github.com/labstack/echo/v4"
)

type Shops interface {
	GetAllShops(c echo.Context) error
	GetShopsBy(c echo.Context) error
}

type shops struct {
	app ports.ShopsApp
}

func NewShopsHandler(app ports.ShopsApp) Shops {
	return &shops{app}
}

// GetAllShops godoc
// @Tags Shops
// @Summary Para que no me fastidies de cual es la ruta
// @Produce json
// @Param lat query number true "Latitud del usuario .ej: 4.678034"
// @Param lon query number true "Longitud del usuario .ej: -74.0496399"
// @Success 200 {object} []dtos.ShopResponse
// @Failure 404
// @Failure 500
// @Router /shops/all [get]
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
		switch err.Error() {
		case "not found":
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
		}
	}

	return c.JSON(http.StatusOK, shops)
}

// GetShopsBy godoc
// @Tags Shops
// @Summary End Point para obtener un negocio con todos sus products, se debe enviar alguno de los 2 query params requeridos.
// @Produce json
// @Param id query string false "Este es el ID del negocio, viene en formato UUID"
// @Param tag query string false "Este es el tag del negocio .ej:dmo"
// @Param lat query number true "Latitud del usuario .ej: 4.678034"
// @Param lon query number true "Longitud del usuario .ej: -74.0496399"
// @Success 200 {object} []dtos.ShopResponse
// @Failure 404
// @Failure 500
// @Router /shops [get]
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
		switch err.Error() {
		case "not found":
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(http.StatusOK, products)
}
