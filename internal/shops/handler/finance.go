package handler

import (
	"net/http"
	"strconv"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	_ "Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsPorts "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/labstack/echo/v4"
)

type ShopFinanceHandler interface {
	GetBalance(c echo.Context) error
	ListTransactions(c echo.Context) error
}

type shopFinanceHandler struct {
	svc shopsPorts.ShopFinanceApp
}

func NewShopFinanceHandler(svc shopsPorts.ShopFinanceApp) ShopFinanceHandler {
	return &shopFinanceHandler{svc}
}

// GetBalance godoc
//
// @Summary      Balance del local
// @Description  Retorna balance_bs y balance_usd (tasa BCV vigente en bcv_rate) y los últimos 10 movimientos (amount_bs y amount_usd).
// @Tags         Shops Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.ShopBalanceResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/balance [get]
func (h *shopFinanceHandler) GetBalance(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	resp, err := h.svc.GetBalance(ctx, *session.ShopID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// ListTransactions godoc
//
// @Summary      Historial de movimientos del local
// @Description  Retorna el historial paginado de movimientos (amount_bs y amount_usd).
// @Tags         Shops Finance
// @Produce      json
// @Param        page   query  int  false  "Página (default 1)"
// @Param        limit  query  int  false  "Límite por página, máx 50 (default 20)"
// @Security     SessionCookie
// @Success      200  {object}  dtos.ShopTransactionsResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/transactions [get]
func (h *shopFinanceHandler) ListTransactions(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}

	resp, err := h.svc.ListTransactions(ctx, *session.ShopID, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}
