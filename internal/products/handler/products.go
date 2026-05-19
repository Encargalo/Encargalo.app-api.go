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
	SearchFlavorsByCategoryID(e echo.Context) error
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
// @Summary      Registrar un nuevo producto
// @Description  Registra un nuevo producto en la tienda asociada al empleado autenticado. Si se envía `flavor_ids`, el producto se marcará automáticamente como con sabores y se asociarán los sabores indicados.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     Cookie
// @Param        item  body      dtos.RegisterItem  true  "Datos del producto a registrar"
// @Success      201   {string}  string             "product registered successfully"
// @Failure      400   {string}  string             "JSON inválido o campos faltantes"
// @Failure      401   {string}  string             "No autorizado"
// @Failure      422   {string}  string             "No se encontró la cuenta del empleado"
// @Failure      500   {string}  string             "Error interno del servidor"
// @Router       /products [post]
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
// @Summary      Actualizar un producto
// @Description  Actualiza la información de un producto existente. Solo puede actualizarlo el Order Manager de la tienda a la que pertenece. La imagen solo se sube a Cloudinary si es diferente a la actual (comparación por hash). Si se envía `flavor_ids` (incluso vacío), los sabores se reemplazan completamente; si no se envía, los sabores no se modifican.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     Cookie
// @Param        item_id  path      string           true  "ID del producto"  Format(uuid)
// @Param        body     body      dtos.UpdateItem  true  "Datos del producto a actualizar"
// @Success      200      {string}  string           "item update success"
// @Failure      400      {string}  string           "item_id inválido o JSON mal formado"
// @Failure      401      {string}  string           "No autorizado"
// @Failure      404      {string}  string           "Producto no encontrado"
// @Failure      500      {string}  string           "Error interno del servidor"
// @Router       /products/{item_id} [put]
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

// SearchProductsByShopID godoc
// @Summary      Obtener productos por tienda
// @Description  Retorna todos los productos agrupados por categoría de una tienda. Solo incluye productos disponibles, ordenados por precio ascendente.
// @Tags         Products
// @Produce      json
// @Param        shop_id  query     string  true  "ID de la tienda"  Format(uuid)
// @Success      200      {object}  dtos.CategoriesResponse
// @Failure      400      {string}  string  "shop_id es requerido o inválido"
// @Failure      404      {string}  string  "Productos no encontrados"
// @Failure      500      {string}  string  "Error interno del servidor"
// @Router       /products [get]
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

// SearchAdditionsByShopID godoc
// @Summary      Obtener adiciones por categoría
// @Description  Retorna todas las adiciones disponibles de una categoría, ordenadas por precio ascendente.
// @Tags         Products
// @Produce      json
// @Param        category_id  query     string  true  "ID de la categoría"  Format(uuid)
// @Success      200          {object}  dtos.AdditionsResponse
// @Failure      400          {string}  string  "category_id es requerido o inválido"
// @Failure      404          {string}  string  "Adiciones no encontradas"
// @Failure      500          {string}  string  "Error interno del servidor"
// @Router       /products/additions [get]
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

// SearchFlavorsByItemID godoc
// @Summary      Obtener sabores por producto
// @Description  Retorna todos los sabores disponibles asociados a un producto específico.
// @Tags         Products
// @Produce      json
// @Param        item_id  query     string  true  "ID del producto" Format(uuid)
// @Success      200      {object}  dtos.FlavorsResponse
// @Failure      400      {string}  string  "item_id es requerido o inválido"
// @Failure      404      {string}  string  "Sabores no encontrados"
// @Failure      500      {string}  string  "Error interno del servidor"
// @Router       /products/flavors [get]
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

// SearchFlavorsByCategoryID godoc
// @Summary      Obtener sabores por categoría
// @Description  Retorna todos los sabores disponibles asociados a una categoría específica.
// @Tags         Products
// @Produce      json
// @Param        category_id  query     string  true  "ID de la categoría" Format(uuid)
// @Success      200          {object}  dtos.FlavorsResponse
// @Failure      400          {string}  string  "category_id es requerido o inválido"
// @Failure      404          {string}  string  "Sabores no encontrados"
// @Failure      500          {string}  string  "Error interno del servidor"
// @Router       /products/flavors/category [get]
func (p *productsHandler) SearchFlavorsByCategoryID(e echo.Context) error {

	ctx := e.Request().Context()

	categoryIDParam := e.QueryParam("category_id")
	if categoryIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "category_id es requerido")
	}

	categoryID, err := uuid.Parse(categoryIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "category_id inválido")
	}

	flavors, err := p.svc.SearchFlavorsByCategoryID(ctx, categoryID)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, flavors)
}

// SearchBestSellersByShopID godoc
// @Summary      Obtener los productos mas vendidos por tienda
// @Description  Retorna todos los productos más vendidos de una tienda específica, ordenados por ventas descendente.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        shop_id  query     string  true  "ID de la tienda" Format(uuid)
// @Success      200      {object}  dtos.ItemsResponse
// @Failure      400      {string}  string  "shop_id es requerido o inválido"
// @Failure      404      {string}  string  "Productos más vendidos no encontrados"
// @Failure      500      {string}  string  "Error interno del servidor"
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
