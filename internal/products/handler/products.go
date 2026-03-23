package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	emplService "Encargalo.app-api.go/internal/employee/domain/ports"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProductsHandler interface {
	RegisterItems(e echo.Context) error
	UpdateItem(e echo.Context) error
	SearchProductsByShopID(e echo.Context) error
	SearchAdditionsByShopID(e echo.Context) error
	SearchFlavorsByItemID(e echo.Context) error
	SearchBestSellersByShopID(e echo.Context) error
}

type productsHandler struct {
	svc         ports.ProductsApp
	emplService emplService.EmployeeApp
}

func NewProducsHandler(svc ports.ProductsApp, emplService emplService.EmployeeApp) ProductsHandler {
	return &productsHandler{svc, emplService}
}

// RegisterItems godoc
// @Summary Registrar un nuevo producto
// @Description Registra un nuevo producto en la tienda asociada al empleado autenticado.
// @Tags Products
// @Accept json
// @Produce json
// @Security Cookie
// @Param item body dtos.RegisterItem true "Datos del producto a registrar"
// @Success 201 {string} string "product registered successfully"
// @Failure 400 {string} string "JSON inválido o mal formado"
// @Failure 401 {string} string "No autorizado"
// @Failure 422 {string} string "No se encontró la cuenta del empleado"
// @Failure 500 {string} string "Error interno del servidor"
// @Router /products [post]
func (p *productsHandler) RegisterItems(e echo.Context) error {

	ctx := e.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var item dtos.RegisterItem

	if err := e.Bind(&item); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := item.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, session.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "No se encontró la cuenta que quiere registrar el item")
	}

	item.ShopID = empl.ShopID

	if err := p.svc.RegisterItem(ctx, item); err != nil {
		return err
	}

	return e.JSON(http.StatusCreated, "product registered successfully")
}

// UpdateItem godoc
// @Summary      Actualizar un item
// @Description  Actualiza la información de un producto existente.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        item_id  path      string              true  "ID del item"
// @Param        body     body      dtos.UpdateItem     true  "Datos del item a actualizar"
// @Success      200      {string}  string              "item update success"
// @Failure      400      {object}  map[string]string   "Bad Request"
// @Failure      401      {object}  map[string]string   "Unauthorized"
// @Failure      500      {object}  map[string]string   "Internal Server Error"
// @Router       /items/{item_id} [put]
func (p *productsHandler) UpdateItem(e echo.Context) error {

	ctx := e.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	itemIDParam := e.Param("item_id")

	itemID, err := uuid.Parse(itemIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "item_id inválido")
	}

	var item dtos.UpdateItem

	if err := e.Bind(&item); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := item.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := p.svc.UpdateItem(ctx, item, itemID, session.UserID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "item update success")
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
		return err
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
		return err
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
		return err
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
// @Router       /products/best-sellers [get]
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
		return err
	}

	return e.JSON(http.StatusOK, topSeller)
}
