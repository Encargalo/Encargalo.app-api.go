package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/products/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProductsHandler interface {
	SearchProductsByShopID(e echo.Context) error
	SearchAdditionsByShopID(e echo.Context) error
	SearchFlavorsByItemID(e echo.Context) error
	SearchBestSellersByShopID(e echo.Context) error
}

type productsHandler struct {
	svc ports.ProductsApp
}

func NewProducsHandler(svc ports.ProductsApp) ProductsHandler {
	return &productsHandler{svc}
}

func (p *productsHandler) SearchProductsByShopID(e echo.Context) error {

	ctx := e.Request().Context()

	shopIDParam := e.QueryParam("shop_id")
	if shopIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shop_id es requerido")
	}

	shopID, err := uuid.Parse(shopIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shop_id inválido")
	}

	items, err := p.svc.SearchProductsByShopID(ctx, shopID)
	if err != nil {
		switch err.Error() {
		case "products not found":
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return e.JSON(http.StatusOK, items)
}

func (p *productsHandler) SearchAdditionsByShopID(e echo.Context) error {

	ctx := e.Request().Context()

	categoryIDParam := e.QueryParam("category_id")
	if categoryIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "category_id es requerido")
	}

	categoryID, err := uuid.Parse(categoryIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "category_id inválido")
	}

	additions, err := p.svc.SearchAdditionsByShopID(ctx, categoryID)
	if err != nil {
		switch err.Error() {
		case "additions not found":
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return e.JSON(http.StatusOK, additions)
}

func (p *productsHandler) SearchFlavorsByItemID(e echo.Context) error {

	ctx := e.Request().Context()

	itemIDParam := e.QueryParam("item_id")
	if itemIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "item_id es requerido")
	}

	itemID, err := uuid.Parse(itemIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "item_id inválido")
	}

	flavors, err := p.svc.SearchFlavorsByItemID(ctx, itemID)
	if err != nil {
		switch err.Error() {
		case "flavors not found":
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return e.JSON(http.StatusOK, flavors)

}

// SearchBestSellersByShopID godoc
// @Summary      Obtener los productos mas vendidos por tienda
// @Description  Esta función maneja la solicitud para obtener los productos mas vendidos por tienda de una tienda específica.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        shop_id  query     string  true  "ID de la tienda" Format(uuid)
// @Success      200  {object}   dtos.ItemsResponse
// @Failure      400  {string}  string  "shop_id es requerido o inválido"
// @Failure      404  {string}  string  "Productos más vendidos no encontrados"
// @Failure      500  {string}  string  "Error interno del servidor
// @Router       /best-sellers [get]
func (p *productsHandler) SearchBestSellersByShopID(e echo.Context) error {

	ctx := e.Request().Context()

	shopIDParam := e.QueryParam("shop_id")
	if shopIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shop_id is required")
	}

	shopID, err := uuid.Parse(shopIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shop_id invalid")
	}

	topSeller, err := p.svc.SearchBestSellersByShopID(ctx, shopID)
	if err != nil {
		switch err.Error() {
		case "best sellers not found":
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return e.JSON(http.StatusOK, topSeller)
}
