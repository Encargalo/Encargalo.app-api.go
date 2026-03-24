package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	employeedto "Encargalo.app-api.go/internal/employee/domain/dto"
	mocksempl "Encargalo.app-api.go/internal/mocks/employee/domain/ports"
	mocksprod "Encargalo.app-api.go/internal/mocks/products/domain/ports"
	"Encargalo.app-api.go/internal/pkg/pagination"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	productshandler "Encargalo.app-api.go/internal/products/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func withSession(tc buildhttptest.ControllerCase, session authmodels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

// --- RegisterItems ---

func TestRegisterItems_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza Hawaiana",
		"price":       32000,
		"image":       "data:image/jpeg;base64,/9j/abc123",
		"description": "Pizza con piña y jamón",
	})
	tc := buildhttptest.SetupControllerCase("/products", http.MethodPost, body)
	tc = withSession(tc, session)

	emplResp := employeedto.EmployeeResponse{ID: userID, ShopID: shopID}
	empl.On("SearchEmployeeByID", tc.Req.Context(), userID).Return(emplResp, nil)
	svc.On("RegisterItem", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItem")).Return(nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.RegisterItems(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterItems_NoSession(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza",
		"price":       15000,
		"image":       "data:image/jpeg;base64,/9j/abc",
		"description": "Descripción",
	})
	tc := buildhttptest.SetupControllerCase("/products", http.MethodPost, body)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.RegisterItems(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestRegisterItems_InvalidBody(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	tc := buildhttptest.SetupControllerCase("/products", http.MethodPost, bytes.NewBufferString("{invalid}"))
	tc = withSession(tc, session)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.RegisterItems(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItems_ValidationFail(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	// Missing required fields: name, price, image, description
	body := buildBody(t, map[string]any{"category_id": uuid.New().String()})
	tc := buildhttptest.SetupControllerCase("/products", http.MethodPost, body)
	tc = withSession(tc, session)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.RegisterItems(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItems_EmployeeNotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}
	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza",
		"price":       15000,
		"image":       "data:image/jpeg;base64,/9j/abc",
		"description": "Descripción",
	})
	tc := buildhttptest.SetupControllerCase("/products", http.MethodPost, body)
	tc = withSession(tc, session)

	empl.On("SearchEmployeeByID", tc.Req.Context(), userID).Return(employeedto.EmployeeResponse{}, errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.RegisterItems(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestRegisterItems_ServiceError(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}
	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza",
		"price":       15000,
		"image":       "data:image/jpeg;base64,/9j/abc",
		"description": "Descripción",
	})
	tc := buildhttptest.SetupControllerCase("/products", http.MethodPost, body)
	tc = withSession(tc, session)

	emplResp := employeedto.EmployeeResponse{ID: userID, ShopID: shopID}
	empl.On("SearchEmployeeByID", tc.Req.Context(), userID).Return(emplResp, nil)
	svc.On("RegisterItem", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItem")).Return(errcustom.ErrUnexpectedError)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.RegisterItems(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- UpdateItem ---

func TestUpdateItem_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":        "Pizza actualizada",
		"price":       35000,
		"description": "Nueva descripción",
	})
	tc := buildhttptest.SetupControllerCase("/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	svc.On("UpdateItem", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItem"), itemID, userID).Return(nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.UpdateItem(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateItem_NoSession(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	itemID := uuid.New()
	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/products/"+itemID.String(), http.MethodPut, body)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestUpdateItem_InvalidItemID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/products/not-a-uuid", http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateItem_ValidationFail(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	// Missing required fields: price, description
	body := buildBody(t, map[string]any{"name": "Pizza"})
	tc := buildhttptest.SetupControllerCase("/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateItem_NotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	svc.On("UpdateItem", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItem"), itemID, userID).Return(errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestUpdateItem_UnexpectedError(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	svc.On("UpdateItem", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItem"), itemID, userID).Return(errcustom.ErrUnexpectedError)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- SearchProductsByShopID ---

func TestSearchProductsByShopID_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	shopID := uuid.New()
	expected := pagination.Response[dtos.CategoriesResponse]{Data: dtos.CategoriesResponse{}}

	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 10}).Return(expected, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchProductsByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchProductsByShopID_MissingShopID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products", http.MethodGet, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchProductsByShopID_InvalidShopID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products?shop_id=not-a-uuid", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", "not-a-uuid")

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchProductsByShopID_NotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 10}).
		Return(pagination.Response[dtos.CategoriesResponse]{}, errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestSearchProductsByShopID_UnexpectedError(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 10}).
		Return(pagination.Response[dtos.CategoriesResponse]{}, errcustom.ErrUnexpectedError)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- SearchAdditionsByShopID ---

func TestSearchAdditionsByShopID_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	categoryID := uuid.New()
	expected := pagination.Response[dtos.AdditionsResponse]{Data: dtos.AdditionsResponse{}}

	tc := buildhttptest.SetupControllerCase("/products/additions?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchAdditionsByShopID", tc.Req.Context(), categoryID, pagination.Params{Page: 1, Limit: 20}).Return(expected, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchAdditionsByShopID_MissingCategoryID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/additions", http.MethodGet, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchAdditionsByShopID_InvalidCategoryID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/additions?category_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", "bad")

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchAdditionsByShopID_NotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	categoryID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/additions?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchAdditionsByShopID", tc.Req.Context(), categoryID, pagination.Params{Page: 1, Limit: 20}).
		Return(pagination.Response[dtos.AdditionsResponse]{}, errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

// --- SearchFlavorsByItemID ---

func TestSearchFlavorsByItemID_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	itemID := uuid.New()
	expected := pagination.Response[dtos.FlavorsResponse]{Data: dtos.FlavorsResponse{}}

	tc := buildhttptest.SetupControllerCase("/products/flavors?item_id="+itemID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", itemID.String())

	svc.On("SearchFlavorsByItemID", tc.Req.Context(), itemID, pagination.Params{Page: 1, Limit: 20}).Return(expected, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchFlavorsByItemID_MissingItemID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors", http.MethodGet, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByItemID_InvalidItemID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors?item_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", "bad")

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByItemID_NotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	itemID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/flavors?item_id="+itemID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", itemID.String())

	svc.On("SearchFlavorsByItemID", tc.Req.Context(), itemID, pagination.Params{Page: 1, Limit: 20}).
		Return(pagination.Response[dtos.FlavorsResponse]{}, errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

// --- SearchFlavorsByCategoryID ---

func TestSearchFlavorsByCategoryID_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	categoryID := uuid.New()
	expected := pagination.Response[dtos.FlavorsResponse]{Data: dtos.FlavorsResponse{}}

	tc := buildhttptest.SetupControllerCase("/products/flavors/category?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchFlavorsByCategoryID", tc.Req.Context(), categoryID, pagination.Params{Page: 1, Limit: 20}).Return(expected, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchFlavorsByCategoryID_MissingCategoryID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors/category", http.MethodGet, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByCategoryID_InvalidCategoryID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors/category?category_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", "bad")

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByCategoryID_NotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	categoryID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/flavors/category?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchFlavorsByCategoryID", tc.Req.Context(), categoryID, pagination.Params{Page: 1, Limit: 20}).
		Return(pagination.Response[dtos.FlavorsResponse]{}, errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

// --- SearchBestSellersByShopID ---

func TestSearchBestSellersByShopID_Success(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	shopID := uuid.New()
	expected := pagination.Response[dtos.ItemsResponse]{Data: dtos.ItemsResponse{}}

	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchBestSellersByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 5}).Return(expected, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchBestSellersByShopID_MissingShopID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/best-sellers", http.MethodGet, nil)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchBestSellersByShopID_InvalidShopID(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", "bad")

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchBestSellersByShopID_NotFound(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchBestSellersByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 5}).
		Return(pagination.Response[dtos.ItemsResponse]{}, errcustom.ErrNotFound)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestSearchBestSellersByShopID_UnexpectedError(t *testing.T) {
	svc := mocksprod.NewProductsApp(t)
	empl := mocksempl.NewEmployeeApp(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchBestSellersByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 5}).
		Return(pagination.Response[dtos.ItemsResponse]{}, errcustom.ErrUnexpectedError)

	h := productshandler.NewProducsHandler(svc, empl)
	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}
