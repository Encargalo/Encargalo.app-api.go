package handler

import (
	"log/slog"
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
	SearchShops(c echo.Context) error
	SetOpenClose(c echo.Context) error
	GetTurboEntry(c echo.Context) error
}

type shops struct {
	app ports.ShopsApp
}

func NewShopsHandler(app ports.ShopsApp) Shops {
	return &shops{app}
}

// SetOpenClose godoc
// @Summary      Abrir o cerrar el local
// @Description  Cambia el estado de apertura del local asociado a la sesión del owner. Al abrir (`open=true`) se crea un lote de jornada si no hay uno abierto. Al cerrar solo se marca el local como cerrado: el lote sigue abierto para gestionar órdenes activas y se cierra automáticamente cuando ya no quedan. La primera vez que un local se abre se activan automáticamente sus ventanas de promoción de lanzamiento: 24h con 50% de descuento subsidiado por la plataforma y 192h (8 días) con comisión 0%.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body      object{open=bool}  true  "Estado deseado"
// @Success      204  "Estado actualizado"
// @Failure      400  {string}  string  "Body inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/shops/me/status [patch]
func (s *shops) SetOpenClose(c echo.Context) error {
	ctx := c.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	var req struct {
		Open bool `json:"open"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.app.SetOpenClose(ctx, session.ShopID, req.Open); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
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

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	shop, err := s.app.GetShopByID(ctx, *session.ShopID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, shop)
}

// SearchShops godoc
// @Summary      Buscar tiendas por texto y proximidad
// @Description  Retorna tiendas que coincidan con la búsqueda en nombre de tienda, categoría o macro, dentro de 3 km (2 km si type=Turbo). matched_products incluye sabores y adiciones disponibles del item (details). Query opcional `type` (`Restaurant`|`Supermarket`|`Turbo`); si se omite, incluye Restaurant y Supermarket (Turbo se descubre vía /turbo/entry).
// @Tags         Shops
// @Produce      json
// @Security     SessionCookie
// @Param        q     query     string  true   "Texto a buscar, ej: pizza"
// @Param        lat   query     number  true   "Latitud del usuario"
// @Param        lon   query     number  true   "Longitud del usuario"
// @Param        type  query     string  false  "Filtro de tipo: Restaurant, Supermarket o Turbo"
// @Success      200  {array}   dtos.SearchShopResponse  "Tiendas encontradas"
// @Failure      400  {string}  string                   "Parámetros inválidos o faltantes"
// @Failure      401  {string}  string                   "No autorizado"
// @Failure      500  {string}  string                   "Error interno del servidor"
// @Router       /shops/search [get]
func (s *shops) SearchShops(c echo.Context) error {
	ctx := c.Request().Context()

	query := dtos.SearchShopsQuery{}
	if err := c.Bind(&query); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := query.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := s.app.SearchShops(ctx, query)
	if err != nil {
		slog.Error("error en SearchShops", "error", err, "query", query.Q, "lat", query.Latitude, "lon", query.Longitude, "type", query.Type)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// GetAllShops godoc
// @Summary      Obtener todas las tiendas cercanas
// @Description  Retorna todas las tiendas disponibles ordenadas por proximidad a las coordenadas proporcionadas, dentro de 3 km (2 km si type=Turbo). Query opcional `type` (`Restaurant`|`Supermarket`|`Turbo`); si se omite, incluye Restaurant y Supermarket (Turbo se descubre vía /turbo/entry).
// @Tags         Shops
// @Produce      json
// @Security     SessionCookie
// @Param        lat   query     number  true   "Latitud del usuario, ej: 4.678034"
// @Param        lon   query     number  true   "Longitud del usuario, ej: -74.0496399"
// @Param        type  query     string  false  "Filtro de tipo: Restaurant, Supermarket o Turbo"
// @Success      200  {array}   dtos.ShopResponse  "Lista de tiendas"
// @Failure      400  {string}  string             "Coordenadas inválidas, type inválido o faltantes"
// @Failure      401  {string}  string             "No autorizado"
// @Failure      404  {string}  string             "No se encontraron tiendas"
// @Failure      500  {string}  string             "Error interno del servidor"
// @Router       /shops/all [get]
func (s *shops) GetAllShops(c echo.Context) error {

	ctx := c.Request().Context()

	query := dtos.ListShopsQuery{}

	if err := c.Bind(&query); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := query.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	shops, err := s.app.GetAllShops(ctx, query)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, shops)
}

// GetShopsBy godoc
// @Summary      Obtener tienda por ID o slug
// @Description  Retorna la información de una tienda junto con sus productos. Se debe enviar al menos uno de los parámetros: `id` o `slug`.
// @Tags         Shops
// @Produce      json
// @Security     SessionCookie
// @Param        id    query     string  false  "ID de la tienda (UUID)"
// @Param        slug  query     string  false  "Slug de la tienda, ej: dmo"
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

// GetTurboEntry godoc
// @Summary      Entrada Turbo: asignar nodo por geolocalización
// @Description  Dadas las coordenadas del cliente, asigna el nodo Turbo abierto más cercano dentro de 2 km (estilo Rappi: el cliente no elige local). Si no hay cobertura, retorna available=false.
// @Tags         Shops
// @Produce      json
// @Security     SessionCookie
// @Param        lat  query     number  true  "Latitud del usuario"
// @Param        lon  query     number  true  "Longitud del usuario"
// @Success      200  {object}  dtos.TurboEntryResponse
// @Failure      400  {string}  string  "Coordenadas inválidas"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /turbo/entry [get]
func (s *shops) GetTurboEntry(c echo.Context) error {
	ctx := c.Request().Context()

	coords := dtos.Coords{}
	if err := c.Bind(&coords); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := coords.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	entry, err := s.app.GetTurboEntry(ctx, coords)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, entry)
}
