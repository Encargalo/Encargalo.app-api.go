package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffProductsHandler interface {
	GetShopProducts(e echo.Context) error
	SearchCategoriesByShopID(e echo.Context) error
	SearchCategorySummariesByShopID(e echo.Context) error
	SearchMacroCategories(e echo.Context) error
	RegisterCategory(e echo.Context) error
	UpdateCategory(e echo.Context) error
	RegisterItem(e echo.Context) error
	UpdateItem(e echo.Context) error
	RegisterFlavor(e echo.Context) error
	UpdateFlavor(e echo.Context) error
	DeleteFlavor(e echo.Context) error
	RegisterItemDiscount(e echo.Context) error
	UpdateItemDiscount(e echo.Context) error
	DeleteItemDiscount(e echo.Context) error
	SearchAdditionsCatalogByShopID(e echo.Context) error
}

type staffProductsHandler struct {
	svc ports.ProductsApp
}

func NewStaffProductsHandler(svc ports.ProductsApp) StaffProductsHandler {
	return &staffProductsHandler{svc}
}

func shopIDFromPath(e echo.Context) (uuid.UUID, error) {
	shopID, err := uuid.Parse(e.Param("id"))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "id de shop inválido")
	}
	return shopID, nil
}

// GetShopProducts godoc
// @Summary      Obtener productos del negocio (staff)
// @Description  Retorna todos los productos agrupados por categoría del shop indicado.
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path      string  true  "Shop ID"
// @Success      200 {object}  dtos.CategoriesResponse
// @Failure      400 {string}  string  "id inválido"
// @Failure      404 {string}  string  "Productos no encontrados"
// @Failure      500 {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products [get]
func (p *staffProductsHandler) GetShopProducts(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	items, err := p.svc.SearchProductsByShopIDForOwner(e.Request().Context(), shopID)
	if err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, items)
}

// SearchCategoriesByShopID godoc
// @Summary      Listar categorías completas del negocio (staff)
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path      string  true  "Shop ID"
// @Success      200 {object}  dtos.CategoriesResponse
// @Failure      400 {string}  string  "id inválido"
// @Failure      404 {string}  string  "Categorías no encontradas"
// @Failure      500 {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products/categories [get]
func (p *staffProductsHandler) SearchCategoriesByShopID(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	categories, err := p.svc.SearchCategoriesByShopID(e.Request().Context(), shopID)
	if err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, categories)
}

// SearchCategorySummariesByShopID godoc
// @Summary      Listar categorías del negocio en resumen (staff)
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id   path      string  true  "Shop ID"
// @Param        for  query     string  false  "Filtrar categorías elegibles" Enums(flavors,additions)
// @Success      200  {object}  dtos.CategorySummariesResponse
// @Failure      400  {string}  string  "id inválido"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products/categories/summary [get]
func (p *staffProductsHandler) SearchCategorySummariesByShopID(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	categories, err := p.svc.SearchCategorySummariesByShopID(e.Request().Context(), shopID, e.QueryParam("for"))
	if err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, categories)
}

// SearchMacroCategories godoc
// @Summary      Obtener macro categorías (staff)
// @Description  Catálogo de macro categorías. Filtro opcional shop_type (Restaurant|Supermarket|Turbo); sin filtro retorna todas. El shop_id del path no se usa (catálogo global) pero se mantiene por consistencia de ruteo.
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id         path   string  true   "Shop ID"
// @Param        shop_type  query  string  false  "Restaurant|Supermarket|Turbo"
// @Success      200  {object}  dtos.MacroCategoriesResponse
// @Failure      400  {string}  string  "id inválido / shop_type inválido"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products/macro-categories [get]
func (p *staffProductsHandler) SearchMacroCategories(e echo.Context) error {
	if _, err := shopIDFromPath(e); err != nil {
		return err
	}

	macroCategories, err := p.svc.SearchMacroCategories(e.Request().Context(), e.QueryParam("shop_type"))
	if err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, macroCategories)
}

// RegisterCategory godoc
// @Summary      Registrar una categoría (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id        path      string                 true  "Shop ID"
// @Param        category  body      dtos.RegisterCategory  true  "Datos de la categoría a registrar"
// @Success      201       {string}  string                 "category registered successfully"
// @Failure      400       {string}  string                 "JSON inválido, campos faltantes o id inválido"
// @Failure      500       {string}  string                 "Error interno del servidor"
// @Router       /shops/staff/{id}/products/categories [post]
func (p *staffProductsHandler) RegisterCategory(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	var category dtos.RegisterCategory
	if err := e.Bind(&category); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := category.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	category.ShopID = shopID

	if err := p.svc.RegisterCategory(e.Request().Context(), category); err != nil {
		return err
	}

	return response.Send(e, http.StatusCreated, "category registered successfully")
}

// UpdateCategory godoc
// @Summary      Actualizar una categoría (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id           path      string               true  "Shop ID"
// @Param        category_id  path      string               true  "ID de la categoría"  Format(uuid)
// @Param        body         body      dtos.UpdateCategory  true  "Datos de la categoría a actualizar"
// @Success      200          {string}  string               "category update success"
// @Failure      400          {string}  string               "id/category_id inválido o JSON mal formado"
// @Failure      404          {string}  string               "Categoría no encontrada"
// @Failure      500          {string}  string               "Error interno del servidor"
// @Router       /shops/staff/{id}/products/categories/{category_id} [put]
func (p *staffProductsHandler) UpdateCategory(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	categoryID, err := uuid.Parse(e.Param("category_id"))
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

	if err := p.svc.UpdateCategory(e.Request().Context(), category, categoryID, shopID); err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, "category update success")
}

// RegisterItem godoc
// @Summary      Registrar un producto (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path      string             true  "Shop ID"
// @Param        item  body      dtos.RegisterItem  true  "Datos del producto a registrar"
// @Success      201   {string}  string             "product registered successfully"
// @Failure      400   {string}  string             "JSON inválido, campos faltantes o id inválido"
// @Failure      500   {string}  string             "Error interno del servidor"
// @Router       /shops/staff/{id}/products/items [post]
func (p *staffProductsHandler) RegisterItem(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	var item dtos.RegisterItem
	if err := e.Bind(&item); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := item.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	item.ShopID = shopID

	if err := p.svc.RegisterItem(e.Request().Context(), item); err != nil {
		return err
	}

	return response.Send(e, http.StatusCreated, "product registered successfully")
}

// UpdateItem godoc
// @Summary      Actualizar un producto (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id       path      string           true  "Shop ID"
// @Param        item_id  path      string           true  "ID del producto"  Format(uuid)
// @Param        body     body      dtos.UpdateItem  true  "Datos del producto a actualizar"
// @Success      200      {string}  string           "item update success"
// @Failure      400      {string}  string           "id/item_id inválido o JSON mal formado"
// @Failure      404      {string}  string           "Producto no encontrado"
// @Failure      500      {string}  string           "Error interno del servidor"
// @Router       /shops/staff/{id}/products/items/{item_id} [put]
func (p *staffProductsHandler) UpdateItem(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(e.Param("item_id"))
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

	if err := p.svc.UpdateItem(e.Request().Context(), item, itemID, shopID); err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, "item update success")
}

// RegisterFlavor godoc
// @Summary      Registrar un sabor (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id      path      string                true  "Shop ID"
// @Param        flavor  body      dtos.RegisterFlavor  true  "Datos del sabor a registrar"
// @Success      201     {string}  string                "flavor registered successfully"
// @Failure      400     {string}  string                "JSON inválido, campos faltantes o id inválido"
// @Failure      404     {string}  string                "Categoría no encontrada"
// @Failure      500     {string}  string                "Error interno del servidor"
// @Router       /shops/staff/{id}/products/flavors [post]
func (p *staffProductsHandler) RegisterFlavor(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	var flavor dtos.RegisterFlavor
	if err := e.Bind(&flavor); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := flavor.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	flavor.ShopID = shopID

	if err := p.svc.RegisterFlavor(e.Request().Context(), flavor); err != nil {
		return err
	}

	return response.Send(e, http.StatusCreated, "flavor registered successfully")
}

// UpdateFlavor godoc
// @Summary      Actualizar un sabor (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id         path      string             true  "Shop ID"
// @Param        flavor_id  path      string             true  "ID del sabor"  Format(uuid)
// @Param        flavor     body      dtos.UpdateFlavor  true  "Datos del sabor a actualizar"
// @Success      200        {string}  string             "flavor update success"
// @Failure      400        {string}  string             "id/flavor_id inválido o JSON mal formado"
// @Failure      404        {string}  string             "Sabor no encontrado"
// @Failure      500        {string}  string             "Error interno del servidor"
// @Router       /shops/staff/{id}/products/flavors/{flavor_id} [put]
func (p *staffProductsHandler) UpdateFlavor(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	flavorID, err := uuid.Parse(e.Param("flavor_id"))
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

	if err := p.svc.UpdateFlavor(e.Request().Context(), flavor, flavorID, shopID); err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, "flavor update success")
}

// DeleteFlavor godoc
// @Summary      Eliminar un sabor (staff)
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id         path      string  true  "Shop ID"
// @Param        flavor_id  path      string  true  "ID del sabor"  Format(uuid)
// @Success      200        {string}  string  "flavor deleted successfully"
// @Failure      400        {string}  string  "id/flavor_id inválido"
// @Failure      404        {string}  string  "Sabor no encontrado"
// @Failure      500        {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products/flavors/{flavor_id} [delete]
func (p *staffProductsHandler) DeleteFlavor(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	flavorID, err := uuid.Parse(e.Param("flavor_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "flavor_id inválido")
	}

	if err := p.svc.DeleteFlavor(e.Request().Context(), flavorID, shopID); err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, "flavor deleted successfully")
}

// RegisterItemDiscount godoc
// @Summary      Registrar un descuento (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id        path      string                     true  "Shop ID"
// @Param        discount  body      dtos.RegisterItemDiscount  true  "Datos del descuento a registrar"
// @Success      201       {string}  string                     "item discount registered successfully"
// @Failure      400       {string}  string                     "JSON inválido, campos faltantes o id inválido"
// @Failure      404       {string}  string                     "Producto no encontrado"
// @Failure      500       {string}  string                     "Error interno del servidor"
// @Router       /shops/staff/{id}/products/discounts [post]
func (p *staffProductsHandler) RegisterItemDiscount(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	var discount dtos.RegisterItemDiscount
	if err := e.Bind(&discount); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := discount.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	discount.ShopID = shopID

	if err := p.svc.RegisterItemDiscount(e.Request().Context(), discount); err != nil {
		return err
	}

	return response.Send(e, http.StatusCreated, "item discount registered successfully")
}

// UpdateItemDiscount godoc
// @Summary      Actualizar un descuento (staff)
// @Tags         Shops Staff
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id           path      string                   true  "Shop ID"
// @Param        discount_id  path      string                   true  "ID del descuento"  Format(uuid)
// @Param        body         body      dtos.UpdateItemDiscount  true  "Datos del descuento a actualizar"
// @Success      200          {string}  string                   "item discount update success"
// @Failure      400          {string}  string                   "id/discount_id inválido o JSON mal formado"
// @Failure      404          {string}  string                   "Descuento no encontrado"
// @Failure      500          {string}  string                   "Error interno del servidor"
// @Router       /shops/staff/{id}/products/discounts/{discount_id} [put]
func (p *staffProductsHandler) UpdateItemDiscount(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	discountID, err := uuid.Parse(e.Param("discount_id"))
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

	if err := p.svc.UpdateItemDiscount(e.Request().Context(), discount, discountID, shopID); err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, "item discount update success")
}

// DeleteItemDiscount godoc
// @Summary      Eliminar un descuento (staff)
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id           path      string  true  "Shop ID"
// @Param        discount_id  path      string  true  "ID del descuento"  Format(uuid)
// @Success      200          {string}  string  "item discount deleted successfully"
// @Failure      400          {string}  string  "id/discount_id inválido"
// @Failure      404          {string}  string  "Descuento no encontrado"
// @Failure      500          {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products/discounts/{discount_id} [delete]
func (p *staffProductsHandler) DeleteItemDiscount(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	discountID, err := uuid.Parse(e.Param("discount_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "discount_id inválido")
	}

	if err := p.svc.DeleteItemDiscount(e.Request().Context(), discountID, shopID); err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, "item discount deleted successfully")
}

// SearchAdditionsCatalogByShopID godoc
// @Summary      Catálogo de adiciones del negocio (staff)
// @Tags         Shops Staff
// @Produce      json
// @Security     SessionCookie
// @Param        id  path      string  true  "Shop ID"
// @Success      200 {object}  dtos.AdditionsResponse
// @Failure      400 {string}  string  "id inválido"
// @Failure      500 {string}  string  "Error interno del servidor"
// @Router       /shops/staff/{id}/products/additions/catalog [get]
func (p *staffProductsHandler) SearchAdditionsCatalogByShopID(e echo.Context) error {
	shopID, err := shopIDFromPath(e)
	if err != nil {
		return err
	}

	additions, err := p.svc.SearchAdditionsCatalogByShopID(e.Request().Context(), shopID)
	if err != nil {
		return err
	}

	return response.Send(e, http.StatusOK, additions)
}
