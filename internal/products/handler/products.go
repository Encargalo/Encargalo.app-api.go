package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProductsHandler interface {
	SearchProductsByShopID(e echo.Context) error
	SearchAdditionsByItemID(e echo.Context) error
	SearchFlavorsByItemID(e echo.Context) error
	SearchFlavorsByCategoryID(e echo.Context) error
	SearchBestSellersByShopID(e echo.Context) error
	GetMyProducts(e echo.Context) error
	RegisterItem(e echo.Context) error
	UpdateItem(e echo.Context) error
	RegisterCategory(e echo.Context) error
	UpdateCategory(e echo.Context) error
	GetTrendingProducts(e echo.Context) error
	RegisterFlavor(e echo.Context) error
	UpdateFlavor(e echo.Context) error
	DeleteFlavor(e echo.Context) error
	SearchMacroCategories(e echo.Context) error
	RegisterItemDiscount(e echo.Context) error
	UpdateItemDiscount(e echo.Context) error
	DeleteItemDiscount(e echo.Context) error
	SearchCategorySummariesByShopID(e echo.Context) error
	SearchCategoriesByShopID(e echo.Context) error
	SearchAdditionsCatalogByShopID(e echo.Context) error
}

type productsHandler struct {
	svc ports.ProductsApp
}

func NewProducsHandler(svc ports.ProductsApp) ProductsHandler {
	return &productsHandler{svc}
}

// SearchProductsByShopID godoc
// @Summary      Obtener productos por tienda
// @Description  Retorna productos agrupados por categoría para clientes. Excluye categorías `hidden`, items `hidden`, sabores/adiciones no disponibles, y aplica horario de categoría (America/Caracas). Items con `effective_status: out_of_stock` o `can_order: false` se muestran pero no se pueden pedir (p. ej. cuando los sabores activos son menos que `min_selection`).
// @Tags         Products
// @Produce      json
// @Security     SessionCookie
// @Param        shop_id  query     string  true  "ID de la tienda"  Format(uuid)
// @Success      200      {object}  dtos.CategoriesResponse
// @Failure      400      {string}  string  "shop_id es requerido o inválido"
// @Failure      401      {string}  string  "No autorizado"
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

	items, err := p.svc.SearchProductsByShopID(ctx, shopID, true)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, items)
}

// SearchAdditionsByItemID godoc
// @Summary      Obtener adiciones por producto
// @Description  Retorna las adiciones asignadas a un producto específico (subset del catálogo de la categoría), ordenadas como están en items.details.additions.
// @Tags         Products
// @Produce      json
// @Security     SessionCookie
// @Param        item_id  query     string  true  "ID del producto"  Format(uuid)
// @Success      200      {object}  dtos.AdditionsResponse
// @Failure      400      {string}  string  "item_id es requerido o inválido"
// @Failure      401      {string}  string  "No autorizado"
// @Failure      404      {string}  string  "Adiciones no encontradas"
// @Failure      500      {string}  string  "Error interno del servidor"
// @Router       /products/additions [get]
func (p *productsHandler) SearchAdditionsByItemID(e echo.Context) error {

	ctx := e.Request().Context()

	itemIDParam := e.QueryParam("item_id")
	if itemIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "item_id es requerido")
	}

	itemID, err := uuid.Parse(itemIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "item_id inválido")
	}

	additions, err := p.svc.SearchAdditionsByItemID(ctx, itemID)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, additions)
}

// SearchAdditionsCatalogByShopID godoc
// @Summary      Catálogo de adiciones del negocio
// @Description  Retorna los items del negocio cuya categoría tiene macro `adiciones`. Usar al crear/editar un producto para seleccionar `addition_ids`.
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.AdditionsResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/products/additions/catalog [get]
func (p *productsHandler) SearchAdditionsCatalogByShopID(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	additions, err := p.svc.SearchAdditionsCatalogByShopID(ctx, *session.ShopID)
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
// @Security     SessionCookie
// @Param        item_id  query     string  true  "ID del producto" Format(uuid)
// @Success      200      {object}  dtos.FlavorsResponse
// @Failure      400      {string}  string  "item_id es requerido o inválido"
// @Failure      401      {string}  string  "No autorizado"
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
// @Description  Retorna todos los sabores del catálogo JSONB de una categoría (`details.flavors`). Disponible para Customer y para Owner/Order Manager.
// @Tags         Products
// @Produce      json
// @Security     SessionCookie
// @Param        category_id  query     string  true  "ID de la categoría" Format(uuid)
// @Success      200          {object}  dtos.FlavorsResponse
// @Failure      400          {string}  string  "category_id es requerido o inválido"
// @Failure      401          {string}  string  "No autorizado"
// @Failure      404          {string}  string  "Sabores no encontrados"
// @Failure      500          {string}  string  "Error interno del servidor"
// @Router       /products/flavors/category [get]
// @Router       /business/products/flavors/category [get]
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
// @Description  Retorna todos los productos más vendidos de una tienda específica, ordenados por ventas descendente. Excluye productos cuya categoría tiene un horario configurado que no cubre la hora actual (America/Caracas).
// @Tags         Products
// @Produce      json
// @Security     SessionCookie
// @Param        shop_id  query     string  true  "ID de la tienda" Format(uuid)
// @Success      200      {object}  dtos.ItemsResponse
// @Failure      400      {string}  string  "shop_id es requerido o inválido"
// @Failure      401      {string}  string  "No autorizado"
// @Failure      404      {string}  string  "Productos más vendidos no encontrados"
// @Failure      500      {string}  string  "Error interno del servidor"
// @Router       /products/best-sellers [get]
func (p *productsHandler) SearchBestSellersByShopID(e echo.Context) error {

	ctx := e.Request().Context()

	shopIDParam := e.QueryParam("shop_id")
	if shopIDParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shop_id es requerido")
	}

	shopID, err := uuid.Parse(shopIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shop_id inválido")
	}

	topSeller, err := p.svc.SearchBestSellersByShopID(ctx, shopID)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, topSeller)
}

// GetMyProducts godoc
// @Summary      Obtener productos del negocio
// @Description  Retorna todos los productos agrupados por categoría del negocio asociado a la sesión autenticada.
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.CategoriesResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Productos no encontrados"
// @Failure      422  {string}  string  "El usuario no tiene un negocio asociado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/products [get]
func (p *productsHandler) GetMyProducts(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	items, err := p.svc.SearchProductsByShopIDForOwner(ctx, *session.ShopID)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, items)
}

// RegisterCategory godoc
// @Summary      Registrar una nueva categoría
// @Description  Registra una nueva categoría. Puede incluir `flavors` (name/description); el backend genera el `id` de cada sabor y lo guarda en `details.flavors`. Si hay sabores, `selector_type` debe ser `single_select` (cantidad 1 por sabor; min/max = sabores distintos) o `multi_select` (cantidad N del mismo sabor).
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        category  body      dtos.RegisterCategory  true  "Datos de la categoría a registrar"
// @Success      201       {string}  string                 "category registered successfully"
// @Failure      400       {string}  string                 "JSON inválido o campos faltantes"
// @Failure      401       {string}  string                 "No autorizado"
// @Failure      422       {string}  string                 "El usuario no tiene un negocio asociado"
// @Failure      500       {string}  string                 "Error interno del servidor"
// @Router       /business/products/categories [post]
func (p *productsHandler) RegisterCategory(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	var category dtos.RegisterCategory
	if err := e.Bind(&category); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := category.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	category.ShopID = *session.ShopID

	if err := p.svc.RegisterCategory(ctx, category); err != nil {
		return err
	}

	return e.JSON(http.StatusCreated, "category registered successfully")
}

// UpdateCategory godoc
// @Summary      Actualizar una categoría
// @Description  Actualiza el nombre, macro-categoría, `availability_status` (`active` | `hidden` | `out_of_stock`) y/o horario de una categoría existente. Enviar `schedules` reemplaza todo el horario; omitirlo lo deja sin cambios; `[]` elimina el horario.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        category_id  path      string               true  "ID de la categoría"  Format(uuid)
// @Param        body         body      dtos.UpdateCategory  true  "Datos de la categoría a actualizar"
// @Success      200          {string}  string               "category update success"
// @Failure      400          {string}  string               "category_id inválido o JSON mal formado"
// @Failure      401          {string}  string               "No autorizado"
// @Failure      404          {string}  string               "Categoría no encontrada"
// @Failure      500          {string}  string               "Error interno del servidor"
// @Router       /business/products/categories/{category_id} [put]
func (p *productsHandler) UpdateCategory(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	categoryIDParam := e.Param("category_id")
	categoryID, err := uuid.Parse(categoryIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "category_id inválido")
	}

	var category dtos.UpdateCategory
	if err := e.Bind(&category); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := category.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := p.svc.UpdateCategory(ctx, category, categoryID, *session.ShopID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "category update success")
}

// RegisterItem godoc
// @Summary      Registrar un nuevo producto
// @Description  Registra un nuevo producto. Con sabores: `flavor_ids` + opcional `min_selection`/`max_selection` (defaults 1 y cantidad de sabores). `single_select` = sabores distintos (cantidad 1 c/u); `multi_select` = cantidad N del mismo sabor. `addition_ids` opcional: items de la macro Adiciones; selección fija multi_select max 5.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        item  body      dtos.RegisterItem  true  "Datos del producto a registrar"
// @Success      201   {string}  string             "product registered successfully"
// @Failure      400   {string}  string             "JSON inválido o campos faltantes"
// @Failure      401   {string}  string             "No autorizado"
// @Failure      422   {string}  string             "El usuario no tiene un negocio asociado"
// @Failure      500   {string}  string             "Error interno del servidor"
// @Router       /business/products/items [post]
func (p *productsHandler) RegisterItem(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	var item dtos.RegisterItem
	if err := e.Bind(&item); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := item.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	item.ShopID = *session.ShopID

	if err := p.svc.RegisterItem(ctx, item); err != nil {
		return err
	}

	return e.JSON(http.StatusCreated, "product registered successfully")
}

// GetTrendingProducts godoc
// @Summary      Productos más vendidos cerca del usuario
// @Description  Retorna el producto más vendido de cada local abierto dentro de un radio de 1 km de las coordenadas dadas. Máximo 10 resultados, excluyendo productos cuya categoría tiene un horario configurado que no cubre la hora actual (America/Caracas). Cacheado 15 minutos.
// @Tags         Products
// @Produce      json
// @Security     SessionCookie
// @Param        lat  query     number  true  "Latitud del usuario"
// @Param        lon  query     number  true  "Longitud del usuario"
// @Success      200  {object}  dtos.TrendingResponse
// @Failure      400  {string}  string  "lat/lon requeridos o inválidos"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /products/trending [get]
func (p *productsHandler) GetTrendingProducts(e echo.Context) error {
	ctx := e.Request().Context()

	var coords dtos.TrendingCoords
	if err := e.Bind(&coords); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := coords.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := p.svc.GetTrendingByCoords(ctx, coords)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, result)
}

// UpdateItem godoc
// @Summary      Actualizar un producto
// @Description  Actualiza un producto. `availability_status`: active|hidden|out_of_stock. Si es un item de la macro Adiciones, sincroniza name/price/available en los productos que lo referencian. `addition_ids` asigna adiciones (max selección 5).
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        item_id  path      string           true  "ID del producto"  Format(uuid)
// @Param        body     body      dtos.UpdateItem  true  "Datos del producto a actualizar"
// @Success      200      {string}  string           "item update success"
// @Failure      400      {string}  string           "item_id inválido o JSON mal formado"
// @Failure      401      {string}  string           "No autorizado"
// @Failure      404      {string}  string           "Producto no encontrado"
// @Failure      500      {string}  string           "Error interno del servidor"
// @Router       /business/products/{item_id} [put]
func (p *productsHandler) UpdateItem(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
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

	if err := p.svc.UpdateItem(ctx, item, itemID, *session.ShopID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "item update success")
}

// RegisterFlavor godoc
// @Summary      Registrar un nuevo sabor
// @Description  Registra un nuevo sabor asociado a una categoría del negocio autenticado. Un sabor solo puede asignarse a productos que pertenezcan a esa misma categoría.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        flavor  body      dtos.RegisterFlavor  true  "Datos del sabor a registrar"
// @Success      201     {string}  string                "flavor registered successfully"
// @Failure      400     {string}  string                "JSON inválido o campos faltantes"
// @Failure      401     {string}  string                "No autorizado"
// @Failure      404     {string}  string                "Categoría no encontrada"
// @Failure      422     {string}  string                "El usuario no tiene un negocio asociado"
// @Failure      500     {string}  string                "Error interno del servidor"
// @Router       /business/products/flavors [post]
func (p *productsHandler) RegisterFlavor(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	var flavor dtos.RegisterFlavor
	if err := e.Bind(&flavor); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := flavor.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	flavor.ShopID = *session.ShopID

	if err := p.svc.RegisterFlavor(ctx, flavor); err != nil {
		return err
	}

	return e.JSON(http.StatusCreated, "flavor registered successfully")
}

// UpdateFlavor godoc
// @Summary      Actualizar un sabor
// @Description  Actualiza el nombre, descripción y/o disponibilidad (`available`) de un sabor existente del negocio autenticado. Si desactivar sabores deja al item con menos opciones que `min_selection`, el item pasa a no ser pedible automáticamente.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        flavor_id  path      string             true  "ID del sabor"  Format(uuid)
// @Param        body       body      dtos.UpdateFlavor  true  "Datos del sabor a actualizar"
// @Success      200        {string}  string             "flavor update success"
// @Failure      400        {string}  string             "flavor_id inválido o JSON mal formado"
// @Failure      401        {string}  string             "No autorizado"
// @Failure      404        {string}  string             "Sabor no encontrado"
// @Failure      500        {string}  string             "Error interno del servidor"
// @Router       /business/products/flavors/{flavor_id} [put]
func (p *productsHandler) UpdateFlavor(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	flavorIDParam := e.Param("flavor_id")
	flavorID, err := uuid.Parse(flavorIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "flavor_id inválido")
	}

	var flavor dtos.UpdateFlavor
	if err := e.Bind(&flavor); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := flavor.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := p.svc.UpdateFlavor(ctx, flavor, flavorID, *session.ShopID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "flavor update success")
}

// DeleteFlavor godoc
// @Summary      Eliminar un sabor
// @Description  Elimina un sabor del catálogo JSONB de la categoría y lo quita de todos los productos de esa categoría que lo tenían asignado. Si un producto queda sin sabores, has_flavors pasa a false.
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Param        flavor_id  path      string  true  "ID del sabor"  Format(uuid)
// @Success      200        {string}  string  "flavor deleted successfully"
// @Failure      400        {string}  string  "flavor_id inválido"
// @Failure      401        {string}  string  "No autorizado"
// @Failure      404        {string}  string  "Sabor no encontrado"
// @Failure      500        {string}  string  "Error interno del servidor"
// @Router       /business/products/flavors/{flavor_id} [delete]
func (p *productsHandler) DeleteFlavor(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	flavorIDParam := e.Param("flavor_id")
	flavorID, err := uuid.Parse(flavorIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "flavor_id inválido")
	}

	if err := p.svc.DeleteFlavor(ctx, flavorID, *session.ShopID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "flavor deleted successfully")
}

// SearchMacroCategories godoc
// @Summary      Obtener macro categorías
// @Description  Retorna el catálogo de macro categorías (Pizza, Parrilla, Pollo, etc.) usado para clasificar las categorías del negocio autenticado. Filtro opcional shop_type (Restaurant|Supermarket|Turbo); sin filtro retorna todas.
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Param        shop_type  query  string  false  "Restaurant|Supermarket|Turbo"
// @Success      200  {object}  dtos.MacroCategoriesResponse
// @Failure      400  {string}  string  "shop_type inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/products/macro-categories [get]
func (p *productsHandler) SearchMacroCategories(e echo.Context) error {
	ctx := e.Request().Context()
	shopType := e.QueryParam("shop_type")

	macroCategories, err := p.svc.SearchMacroCategories(ctx, shopType)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, macroCategories)
}

// RegisterItemDiscount godoc
// @Summary      Registrar un descuento propio del negocio
// @Description  Registra un descuento (porcentaje o monto fijo) sobre un producto del negocio autenticado. Solo puede existir un descuento activo por producto. Este descuento se compone con la promo de lanzamiento de la plataforma: se aplica primero sobre el precio de catálogo, y luego, si el negocio está en su ventana de día 1, la promo se aplica sobre ese precio ya rebajado.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        discount  body      dtos.RegisterItemDiscount  true  "Datos del descuento a registrar"
// @Success      201       {string}  string                     "item discount registered successfully"
// @Failure      400       {string}  string                     "JSON inválido, campos faltantes o valor de descuento inválido"
// @Failure      401       {string}  string                     "No autorizado"
// @Failure      404       {string}  string                     "Producto no encontrado"
// @Failure      422       {string}  string                     "El usuario no tiene un negocio asociado"
// @Failure      500       {string}  string                     "Error interno del servidor"
// @Router       /business/products/discounts [post]
func (p *productsHandler) RegisterItemDiscount(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	var discount dtos.RegisterItemDiscount
	if err := e.Bind(&discount); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := discount.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	discount.ShopID = *session.ShopID

	if err := p.svc.RegisterItemDiscount(ctx, discount); err != nil {
		return err
	}

	return e.JSON(http.StatusCreated, "item discount registered successfully")
}

// UpdateItemDiscount godoc
// @Summary      Actualizar un descuento propio del negocio
// @Description  Actualiza el tipo, valor, vigencia o estado activo de un descuento existente del negocio autenticado.
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        discount_id  path      string                   true  "ID del descuento"  Format(uuid)
// @Param        body         body      dtos.UpdateItemDiscount  true  "Datos del descuento a actualizar"
// @Success      200          {string}  string                   "item discount update success"
// @Failure      400          {string}  string                   "discount_id inválido, JSON mal formado o valor de descuento inválido"
// @Failure      401          {string}  string                   "No autorizado"
// @Failure      404          {string}  string                   "Descuento no encontrado"
// @Failure      500          {string}  string                   "Error interno del servidor"
// @Router       /business/products/discounts/{discount_id} [put]
func (p *productsHandler) UpdateItemDiscount(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	discountIDParam := e.Param("discount_id")
	discountID, err := uuid.Parse(discountIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "discount_id inválido")
	}

	var discount dtos.UpdateItemDiscount
	if err := e.Bind(&discount); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := discount.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := p.svc.UpdateItemDiscount(ctx, discount, discountID, *session.ShopID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "item discount update success")
}

// DeleteItemDiscount godoc
// @Summary      Eliminar un descuento propio del negocio
// @Description  Elimina (soft delete) un descuento de producto del negocio autenticado.
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Param        discount_id  path      string  true  "ID del descuento"  Format(uuid)
// @Success      200          {string}  string  "item discount deleted successfully"
// @Failure      400          {string}  string  "discount_id inválido"
// @Failure      401          {string}  string  "No autorizado"
// @Failure      404          {string}  string  "Descuento no encontrado"
// @Failure      500          {string}  string  "Error interno del servidor"
// @Router       /business/products/discounts/{discount_id} [delete]
func (p *productsHandler) DeleteItemDiscount(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	discountIDParam := e.Param("discount_id")
	discountID, err := uuid.Parse(discountIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "discount_id inválido")
	}

	if err := p.svc.DeleteItemDiscount(ctx, discountID, *session.ShopID); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, "item discount deleted successfully")
}

// SearchCategoriesByShopID godoc
// @Summary      Listar categorías completas del negocio
// @Description  Retorna todas las categorías del negocio autenticado con `details` (flavors, selector_type), `availability_status`, `macro_category_id` y `schedules`. No incluye items (usar `GET /business/products` para el catálogo con productos).
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.CategoriesResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Categorías no encontradas"
// @Failure      422  {string}  string  "El usuario no tiene un negocio asociado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/products/categories [get]
func (p *productsHandler) SearchCategoriesByShopID(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	categories, err := p.svc.SearchCategoriesByShopID(ctx, *session.ShopID)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, categories)
}

// SearchCategorySummariesByShopID godoc
// @Summary      Listar categorías del negocio (resumen)
// @Description  Retorna solo id y nombre de cada micro categoría del negocio. Use `for=flavors` o `for=additions` (macro Adiciones).
// @Tags         Business
// @Produce      json
// @Security     SessionCookie
// @Param        for  query     string  false  "Filtrar categorías elegibles" Enums(flavors,additions)
// @Success      200  {object}  dtos.CategorySummariesResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      422  {string}  string  "El usuario no tiene un negocio asociado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /business/products/categories/summary [get]
func (p *productsHandler) SearchCategorySummariesByShopID(e echo.Context) error {
	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if session.ShopID == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "el usuario no tiene un negocio asociado")
	}

	forParam := e.QueryParam("for")

	categories, err := p.svc.SearchCategorySummariesByShopID(ctx, *session.ShopID, forParam)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, categories)
}
