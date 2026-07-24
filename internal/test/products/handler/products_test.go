package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksprod "Encargalo.app-api.go/internal/mocks/products/domain/ports"
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

func newHandler(t *testing.T) (productshandler.ProductsHandler, *mocksprod.ProductsApp) {
	t.Helper()
	svc := mocksprod.NewProductsApp(t)
	return productshandler.NewProducsHandler(svc), svc
}

func withSession(tc buildhttptest.ControllerCase, session authmodels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

// --- RegisterItem ---

func TestRegisterItem_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}

	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza Hawaiana",
		"price":       32000,
		"image":       "data:image/jpeg;base64,/9j/abc123",
		"description": "Pizza con piña y jamón",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/items", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterItem", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItem")).Return(nil)

	err := h.RegisterItem(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterItem_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza",
		"price":       15000,
		"image":       "data:image/jpeg;base64,/9j/abc",
		"description": "Descripción",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/items", http.MethodPost, body)

	err := h.RegisterItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestRegisterItem_NoShopID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: nil}
	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza",
		"price":       15000,
		"image":       "data:image/jpeg;base64,/9j/abc",
		"description": "Descripción",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/items", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestRegisterItem_InvalidBody(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	tc := buildhttptest.SetupControllerCase("/business/products/items", http.MethodPost, bytes.NewBufferString("{invalid}"))
	tc = withSession(tc, session)

	err := h.RegisterItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItem_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{"category_id": uuid.New().String()})
	tc := buildhttptest.SetupControllerCase("/business/products/items", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItem_ServiceError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Pizza",
		"price":       15000,
		"image":       "data:image/jpeg;base64,/9j/abc",
		"description": "Descripción",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/items", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterItem", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItem")).Return(errcustom.ErrUnexpectedError)

	err := h.RegisterItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- UpdateItem ---

func TestUpdateItem_Success(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":        "Pizza actualizada",
		"price":       35000,
		"description": "Nueva descripción",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	svc.On("UpdateItem", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItem"), itemID, userID).Return(nil)

	err := h.UpdateItem(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateItem_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	itemID := uuid.New()
	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/"+itemID.String(), http.MethodPut, body)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestUpdateItem_InvalidItemID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/not-a-uuid", http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateItem_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{"name": "Pizza"})
	tc := buildhttptest.SetupControllerCase("/business/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateItem_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	svc.On("UpdateItem", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItem"), itemID, userID).Return(errcustom.ErrNotFound)

	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestUpdateItem_UnexpectedError(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	itemID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":        "Pizza",
		"price":       35000,
		"description": "Desc",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/"+itemID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("item_id")
	tc.Ctx.SetParamValues(itemID.String())

	svc.On("UpdateItem", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItem"), itemID, userID).Return(errcustom.ErrUnexpectedError)

	err := h.UpdateItem(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- SearchProductsByShopID ---

func TestSearchProductsByShopID_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, true).Return(dtos.CategoriesResponse{}, nil)

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchProductsByShopID_MissingShopID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products", http.MethodGet, nil)

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchProductsByShopID_InvalidShopID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products?shop_id=not-a-uuid", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", "not-a-uuid")

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchProductsByShopID_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, true).Return(dtos.CategoriesResponse(nil), errcustom.ErrNotFound)

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestSearchProductsByShopID_UnexpectedError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, true).Return(dtos.CategoriesResponse(nil), errcustom.ErrUnexpectedError)

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- SearchAdditionsByItemID ---

func TestSearchAdditionsByItemID_Success(t *testing.T) {
	h, svc := newHandler(t)

	itemID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/additions?item_id="+itemID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", itemID.String())

	svc.On("SearchAdditionsByItemID", tc.Req.Context(), itemID).Return(dtos.AdditionsResponse{}, nil)

	err := h.SearchAdditionsByItemID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchAdditionsByItemID_MissingItemID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/additions", http.MethodGet, nil)

	err := h.SearchAdditionsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchAdditionsByItemID_InvalidItemID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/additions?item_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", "bad")

	err := h.SearchAdditionsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchAdditionsByItemID_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	itemID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/additions?item_id="+itemID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", itemID.String())

	svc.On("SearchAdditionsByItemID", tc.Req.Context(), itemID).Return(dtos.AdditionsResponse(nil), errcustom.ErrNotFound)

	err := h.SearchAdditionsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

// --- SearchFlavorsByItemID ---

func TestSearchFlavorsByItemID_Success(t *testing.T) {
	h, svc := newHandler(t)

	itemID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/flavors?item_id="+itemID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", itemID.String())

	svc.On("SearchFlavorsByItemID", tc.Req.Context(), itemID).Return(dtos.FlavorsResponse{}, nil)

	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchFlavorsByItemID_MissingItemID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors", http.MethodGet, nil)

	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByItemID_InvalidItemID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors?item_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", "bad")

	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByItemID_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	itemID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/flavors?item_id="+itemID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("item_id", itemID.String())

	svc.On("SearchFlavorsByItemID", tc.Req.Context(), itemID).Return(dtos.FlavorsResponse(nil), errcustom.ErrNotFound)

	err := h.SearchFlavorsByItemID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

// --- SearchFlavorsByCategoryID ---

func TestSearchFlavorsByCategoryID_Success(t *testing.T) {
	h, svc := newHandler(t)

	categoryID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/flavors/category?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchFlavorsByCategoryID", tc.Req.Context(), categoryID).Return(dtos.FlavorsResponse{}, nil)

	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchFlavorsByCategoryID_MissingCategoryID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors/category", http.MethodGet, nil)

	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByCategoryID_InvalidCategoryID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/flavors/category?category_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", "bad")

	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchFlavorsByCategoryID_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	categoryID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/flavors/category?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchFlavorsByCategoryID", tc.Req.Context(), categoryID).Return(dtos.FlavorsResponse(nil), errcustom.ErrNotFound)

	err := h.SearchFlavorsByCategoryID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

// --- SearchBestSellersByShopID ---

func TestSearchBestSellersByShopID_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchBestSellersByShopID", tc.Req.Context(), shopID).Return(dtos.ItemsResponse{}, nil)

	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchBestSellersByShopID_MissingShopID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/best-sellers", http.MethodGet, nil)

	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchBestSellersByShopID_InvalidShopID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", "bad")

	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchBestSellersByShopID_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchBestSellersByShopID", tc.Req.Context(), shopID).Return(dtos.ItemsResponse(nil), errcustom.ErrNotFound)

	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestSearchBestSellersByShopID_UnexpectedError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/best-sellers?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchBestSellersByShopID", tc.Req.Context(), shopID).Return(dtos.ItemsResponse(nil), errcustom.ErrUnexpectedError)

	err := h.SearchBestSellersByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- RegisterCategory ---

func TestRegisterCategory_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}

	body := buildBody(t, map[string]any{
		"name":              "Desayuno",
		"macro_category_id": uuid.New().String(),
		"schedules": []map[string]any{
			{"start_time": "06:00:00", "end_time": "11:00:00"},
		},
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterCategory", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterCategory")).Return(nil)

	err := h.RegisterCategory(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterCategory_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"name":              "Desayuno",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, body)

	err := h.RegisterCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestRegisterCategory_NoShopID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: nil}
	body := buildBody(t, map[string]any{
		"name":              "Desayuno",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestRegisterCategory_InvalidBody(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, bytes.NewBufferString("{invalid}"))
	tc = withSession(tc, session)

	err := h.RegisterCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterCategory_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{})
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterCategory_InvalidScheduleFormat(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{
		"name":              "Desayuno",
		"macro_category_id": uuid.New().String(),
		"schedules": []map[string]any{
			{"start_time": "6pm", "end_time": "11:00:00"},
		},
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterCategory_ServiceError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{
		"name":              "Desayuno",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterCategory", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterCategory")).Return(errcustom.ErrUnexpectedError)

	err := h.RegisterCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- UpdateCategory ---

func TestUpdateCategory_Success(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	categoryID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":              "Cena",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/"+categoryID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues(categoryID.String())

	svc.On("UpdateCategory", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateCategory"), categoryID, userID).Return(nil)

	err := h.UpdateCategory(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateCategory_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	categoryID := uuid.New()
	body := buildBody(t, map[string]any{
		"name":              "Cena",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/"+categoryID.String(), http.MethodPut, body)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues(categoryID.String())

	err := h.UpdateCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestUpdateCategory_InvalidCategoryID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	body := buildBody(t, map[string]any{
		"name":              "Cena",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/not-a-uuid", http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.UpdateCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateCategory_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	userID := uuid.New()
	categoryID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/"+categoryID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues(categoryID.String())

	err := h.UpdateCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateCategory_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	categoryID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":              "Cena",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/"+categoryID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues(categoryID.String())

	svc.On("UpdateCategory", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateCategory"), categoryID, userID).Return(errcustom.ErrNotFound)

	err := h.UpdateCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestUpdateCategory_Unauthorized(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	categoryID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":              "Cena",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/"+categoryID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues(categoryID.String())

	svc.On("UpdateCategory", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateCategory"), categoryID, userID).Return(errcustom.ErrUnautorized)

	err := h.UpdateCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

func TestUpdateCategory_UnexpectedError(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	categoryID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"name":              "Cena",
		"macro_category_id": uuid.New().String(),
	})
	tc := buildhttptest.SetupControllerCase("/business/products/categories/"+categoryID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("category_id")
	tc.Ctx.SetParamValues(categoryID.String())

	svc.On("UpdateCategory", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateCategory"), categoryID, userID).Return(errcustom.ErrUnexpectedError)

	err := h.UpdateCategory(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- RegisterFlavor ---

func TestRegisterFlavor_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}

	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Picante",
		"description": "Con salsa picante casera",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterFlavor", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterFlavor")).Return(nil)

	err := h.RegisterFlavor(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterFlavor_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Picante",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors", http.MethodPost, body)

	err := h.RegisterFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestRegisterFlavor_NoShopID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: nil}
	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Picante",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestRegisterFlavor_InvalidBody(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	tc := buildhttptest.SetupControllerCase("/business/products/flavors", http.MethodPost, bytes.NewBufferString("{invalid}"))
	tc = withSession(tc, session)

	err := h.RegisterFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterFlavor_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{"name": "Picante"})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterFlavor_ServiceError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{
		"category_id": uuid.New().String(),
		"name":        "Picante",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterFlavor", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterFlavor")).Return(errcustom.ErrUnautorized)

	err := h.RegisterFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

// --- UpdateFlavor ---

func TestUpdateFlavor_Success(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{"name": "Muy picante"})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	svc.On("UpdateFlavor", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateFlavor"), flavorID, userID).Return(nil)

	err := h.UpdateFlavor(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateFlavor_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	flavorID := uuid.New()
	body := buildBody(t, map[string]any{"name": "Muy picante"})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodPut, body)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	err := h.UpdateFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestUpdateFlavor_InvalidFlavorID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	body := buildBody(t, map[string]any{"name": "Muy picante"})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/not-a-uuid", http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.UpdateFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateFlavor_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	err := h.UpdateFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateFlavor_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{"name": "Muy picante"})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	svc.On("UpdateFlavor", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateFlavor"), flavorID, userID).Return(errcustom.ErrNotFound)

	err := h.UpdateFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestUpdateFlavor_Unauthorized(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{"name": "Muy picante"})
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	svc.On("UpdateFlavor", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateFlavor"), flavorID, userID).Return(errcustom.ErrUnautorized)

	err := h.UpdateFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

// --- DeleteFlavor ---

func TestDeleteFlavor_Success(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	svc.On("DeleteFlavor", tc.Req.Context(), flavorID, userID).Return(nil)

	err := h.DeleteFlavor(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestDeleteFlavor_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	flavorID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodDelete, nil)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	err := h.DeleteFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestDeleteFlavor_InvalidFlavorID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	tc := buildhttptest.SetupControllerCase("/business/products/flavors/not-a-uuid", http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.DeleteFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDeleteFlavor_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	svc.On("DeleteFlavor", tc.Req.Context(), flavorID, userID).Return(errcustom.ErrNotFound)

	err := h.DeleteFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestDeleteFlavor_Unauthorized(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	flavorID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	tc := buildhttptest.SetupControllerCase("/business/products/flavors/"+flavorID.String(), http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("flavor_id")
	tc.Ctx.SetParamValues(flavorID.String())

	svc.On("DeleteFlavor", tc.Req.Context(), flavorID, userID).Return(errcustom.ErrUnautorized)

	err := h.DeleteFlavor(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

// --- SearchMacroCategories ---

func TestSearchMacroCategories_Success(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/business/products/macro-categories", http.MethodGet, nil)

	svc.On("SearchMacroCategories", tc.Req.Context()).Return(dtos.MacroCategoriesResponse{}, nil)

	err := h.SearchMacroCategories(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchMacroCategories_UnexpectedError(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/business/products/macro-categories", http.MethodGet, nil)

	svc.On("SearchMacroCategories", tc.Req.Context()).Return(dtos.MacroCategoriesResponse(nil), errcustom.ErrUnexpectedError)

	err := h.SearchMacroCategories(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- RegisterItemDiscount ---

func TestRegisterItemDiscount_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}

	body := buildBody(t, map[string]any{
		"item_id":       uuid.New().String(),
		"discount_type": "percentage",
		"value":         20,
	})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterItemDiscount", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItemDiscount")).Return(nil)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterItemDiscount_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"item_id":       uuid.New().String(),
		"discount_type": "percentage",
		"value":         20,
	})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, body)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestRegisterItemDiscount_NoShopID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: nil}
	body := buildBody(t, map[string]any{
		"item_id":       uuid.New().String(),
		"discount_type": "percentage",
		"value":         20,
	})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestRegisterItemDiscount_InvalidBody(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, bytes.NewBufferString("{invalid}"))
	tc = withSession(tc, session)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItemDiscount_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{"discount_type": "percentage"})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItemDiscount_InvertedWindow(t *testing.T) {
	h, _ := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{
		"item_id":       uuid.New().String(),
		"discount_type": "percentage",
		"value":         20,
		"starts_at":     "2026-07-18T00:00:00Z",
		"ends_at":       "2026-07-11T00:00:00Z",
	})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, body)
	tc = withSession(tc, session)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterItemDiscount_ServiceError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}
	body := buildBody(t, map[string]any{
		"item_id":       uuid.New().String(),
		"discount_type": "percentage",
		"value":         20,
	})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts", http.MethodPost, body)
	tc = withSession(tc, session)

	svc.On("RegisterItemDiscount", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItemDiscount")).Return(errcustom.ErrUnautorized)

	err := h.RegisterItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

// --- UpdateItemDiscount ---

func TestUpdateItemDiscount_Success(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{
		"discount_type": "fixed",
		"value":         5000,
		"is_active":     true,
	})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	svc.On("UpdateItemDiscount", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItemDiscount"), discountID, userID).Return(nil)

	err := h.UpdateItemDiscount(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateItemDiscount_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	discountID := uuid.New()
	body := buildBody(t, map[string]any{"discount_type": "fixed", "value": 5000})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodPut, body)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	err := h.UpdateItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestUpdateItemDiscount_InvalidDiscountID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	body := buildBody(t, map[string]any{"discount_type": "fixed", "value": 5000})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/not-a-uuid", http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.UpdateItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateItemDiscount_ValidationFail(t *testing.T) {
	h, _ := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	err := h.UpdateItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateItemDiscount_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{"discount_type": "fixed", "value": 5000})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	svc.On("UpdateItemDiscount", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItemDiscount"), discountID, userID).Return(errcustom.ErrNotFound)

	err := h.UpdateItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestUpdateItemDiscount_Unauthorized(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	body := buildBody(t, map[string]any{"discount_type": "fixed", "value": 5000})
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodPut, body)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	svc.On("UpdateItemDiscount", tc.Req.Context(), mock.AnythingOfType("dtos.UpdateItemDiscount"), discountID, userID).Return(errcustom.ErrUnautorized)

	err := h.UpdateItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

// --- DeleteItemDiscount ---

func TestDeleteItemDiscount_Success(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	svc.On("DeleteItemDiscount", tc.Req.Context(), discountID, userID).Return(nil)

	err := h.DeleteItemDiscount(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestDeleteItemDiscount_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	discountID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodDelete, nil)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	err := h.DeleteItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestDeleteItemDiscount_InvalidDiscountID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New()}
	tc := buildhttptest.SetupControllerCase("/business/products/discounts/not-a-uuid", http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.DeleteItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDeleteItemDiscount_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	svc.On("DeleteItemDiscount", tc.Req.Context(), discountID, userID).Return(errcustom.ErrNotFound)

	err := h.DeleteItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestDeleteItemDiscount_Unauthorized(t *testing.T) {
	h, svc := newHandler(t)

	userID := uuid.New()
	discountID := uuid.New()
	session := authmodels.ActiveSession{UserID: userID}

	tc := buildhttptest.SetupControllerCase("/business/products/discounts/"+discountID.String(), http.MethodDelete, nil)
	tc = withSession(tc, session)
	tc.Ctx.SetParamNames("discount_id")
	tc.Ctx.SetParamValues(discountID.String())

	svc.On("DeleteItemDiscount", tc.Req.Context(), discountID, userID).Return(errcustom.ErrUnautorized)

	err := h.DeleteItemDiscount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnautorized, err)
}

// --- SearchCategorySummariesByShopID ---

func TestSearchCategorySummariesByShopID_Success(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}

	tc := buildhttptest.SetupControllerCase("/business/products/categories/summary", http.MethodGet, nil)
	tc = withSession(tc, session)

	svc.On("SearchCategorySummariesByShopID", tc.Req.Context(), shopID, "").Return(dtos.CategorySummariesResponse{
		{ID: uuid.New(), Name: "Parrillas"},
	}, nil)

	err := h.SearchCategorySummariesByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchCategorySummariesByShopID_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/business/products/categories/summary", http.MethodGet, nil)

	err := h.SearchCategorySummariesByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestSearchCategorySummariesByShopID_NoShopID(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: nil}
	tc := buildhttptest.SetupControllerCase("/business/products/categories/summary", http.MethodGet, nil)
	tc = withSession(tc, session)

	err := h.SearchCategorySummariesByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestSearchCategorySummariesByShopID_ServiceError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	session := authmodels.ActiveSession{UserID: uuid.New(), ShopID: &shopID}

	tc := buildhttptest.SetupControllerCase("/business/products/categories/summary", http.MethodGet, nil)
	tc = withSession(tc, session)

	svc.On("SearchCategorySummariesByShopID", tc.Req.Context(), shopID, "").Return(nil, errcustom.ErrUnexpectedError)

	err := h.SearchCategorySummariesByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}
