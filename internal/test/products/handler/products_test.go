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

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID).Return(dtos.CategoriesResponse{}, nil)

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

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID).Return(dtos.CategoriesResponse(nil), errcustom.ErrNotFound)

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrNotFound, err)
}

func TestSearchProductsByShopID_UnexpectedError(t *testing.T) {
	h, svc := newHandler(t)

	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products?shop_id="+shopID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("shop_id", shopID.String())

	svc.On("SearchProductsByShopID", tc.Req.Context(), shopID).Return(dtos.CategoriesResponse(nil), errcustom.ErrUnexpectedError)

	err := h.SearchProductsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

// --- SearchAdditionsByShopID ---

func TestSearchAdditionsByShopID_Success(t *testing.T) {
	h, svc := newHandler(t)

	categoryID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/additions?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchAdditionsByShopID", tc.Req.Context(), categoryID).Return(dtos.AdditionsResponse{}, nil)

	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchAdditionsByShopID_MissingCategoryID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/additions", http.MethodGet, nil)

	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchAdditionsByShopID_InvalidCategoryID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/additions?category_id=bad", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", "bad")

	err := h.SearchAdditionsByShopID(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchAdditionsByShopID_NotFound(t *testing.T) {
	h, svc := newHandler(t)

	categoryID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/products/additions?category_id="+categoryID.String(), http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("category_id", categoryID.String())

	svc.On("SearchAdditionsByShopID", tc.Req.Context(), categoryID).Return(dtos.AdditionsResponse(nil), errcustom.ErrNotFound)

	err := h.SearchAdditionsByShopID(tc.Ctx)

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
