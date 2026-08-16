package handler

import (
	"net/http"
	"strconv"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	financedtos "Encargalo.app-api.go/internal/finance/domain/dtos"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	financePorts "Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	_ "Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsPorts "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffShopFinanceHandler interface {
	ListPayables(c echo.Context) error
	GetBalance(c echo.Context) error
	ListTransactions(c echo.Context) error
	RecordPayout(c echo.Context) error
	RecordAdjustment(c echo.Context) error
}

type staffShopFinanceHandler struct {
	staffSvc financePorts.StaffFinanceApp
	shopSvc  shopsPorts.ShopFinanceApp
}

func NewStaffShopFinanceHandler(staffSvc financePorts.StaffFinanceApp, shopSvc shopsPorts.ShopFinanceApp) StaffShopFinanceHandler {
	return &staffShopFinanceHandler{staffSvc: staffSvc, shopSvc: shopSvc}
}

type shopPayablesResponse struct {
	Items []shopPayableItem `json:"items"`
	Total int               `json:"total"`
}

type shopPayableItem struct {
	ShopID     string  `json:"shop_id"`
	Name       string  `json:"name"`
	BalanceBs  float64 `json:"balance_bs"`
	BalanceUSD float64 `json:"balance_usd"`
	BcvRate    float64 `json:"bcv_rate,omitempty"`
	UpdatedAt  string  `json:"updated_at"`
}

// ListPayables godoc
//
// @Summary      Negocios con saldo por pagar (staff)
// @Description  Lista negocios con balance > 0 en balance_bs y balance_usd (tasa BCV vigente).
// @Tags         Shops Staff Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  shopPayablesResponse
// @Failure      401  {string}  string
// @Failure      500  {string}  string
// @Router       /shops/staff/finance/payables [get]
func (h *staffShopFinanceHandler) ListPayables(c echo.Context) error {
	resp, err := h.staffSvc.ListPayables(c.Request().Context(), financeModels.EntityTypeShop)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, mapShopPayables(resp))
}

// GetBalance godoc
//
// @Summary      Balance de un negocio (staff)
// @Description  Retorna balance_bs, balance_usd y los últimos 10 movimientos del local.
// @Tags         Shops Staff Finance
// @Produce      json
// @Param        shopId  path  string  true  "ID del negocio"
// @Security     SessionCookie
// @Success      200  {object}  dtos.ShopBalanceResponse
// @Failure      401  {string}  string
// @Failure      500  {string}  string
// @Router       /shops/staff/finance/{shopId}/balance [get]
func (h *staffShopFinanceHandler) GetBalance(c echo.Context) error {
	shopID, err := uuid.Parse(c.Param("shopId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shopId inválido")
	}
	resp, err := h.shopSvc.GetBalance(c.Request().Context(), shopID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ListTransactions godoc
//
// @Summary      Movimientos de un negocio (staff)
// @Description  Historial paginado de movimientos del local (amount_bs y amount_usd).
// @Tags         Shops Staff Finance
// @Produce      json
// @Param        shopId  path   string  true   "ID del negocio"
// @Param        page    query  int     false  "Página (default 1)"
// @Param        limit   query  int     false  "Límite por página, máx 50 (default 20)"
// @Security     SessionCookie
// @Success      200  {object}  dtos.ShopTransactionsResponse
// @Failure      401  {string}  string
// @Failure      500  {string}  string
// @Router       /shops/staff/finance/{shopId}/transactions [get]
func (h *staffShopFinanceHandler) ListTransactions(c echo.Context) error {
	shopID, err := uuid.Parse(c.Param("shopId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shopId inválido")
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}
	resp, err := h.shopSvc.ListTransactions(c.Request().Context(), shopID, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// RecordPayout godoc
//
// @Summary      Registrar pago a negocio (staff)
// @Description  Registra un pago manual en Bs (proof_image en base64/data-URL). Convierte a USD con BCV y debita el balance del local.
// @Tags         Shops Staff Finance
// @Accept       json
// @Produce      json
// @Param        shopId  path  string                   true  "ID del negocio"
// @Param        body    body  financedtos.PayoutRequest  true  "Monto en Bs + comprobante"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.ActionResponse
// @Failure      400  {string}  string
// @Failure      401  {string}  string
// @Failure      422  {string}  string
// @Failure      503  {string}  string
// @Router       /shops/staff/finance/{shopId}/payouts [post]
func (h *staffShopFinanceHandler) RecordPayout(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	shopID, err := uuid.Parse(c.Param("shopId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shopId inválido")
	}
	var body financedtos.PayoutRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}

	resp, err := h.staffSvc.RecordPayout(ctx, shopID, session.UserID, financeModels.EntityTypeShop, body.AmountBs, body.Notes, body.ProofImage)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// RecordAdjustment godoc
//
// @Summary      Ajuste manual de balance de negocio (staff)
// @Description  Suma o resta saldo: amount_bs se convierte a USD si el balance es nativo USD. Requiere notes.
// @Tags         Shops Staff Finance
// @Accept       json
// @Produce      json
// @Param        shopId  path  string                       true  "ID del negocio"
// @Param        body    body  financedtos.AdjustmentRequest  true  "Dirección, monto en Bs y motivo"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.ActionResponse
// @Failure      400  {string}  string
// @Failure      401  {string}  string
// @Failure      422  {string}  string
// @Failure      503  {string}  string
// @Router       /shops/staff/finance/{shopId}/adjustments [post]
func (h *staffShopFinanceHandler) RecordAdjustment(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	shopID, err := uuid.Parse(c.Param("shopId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "shopId inválido")
	}
	var body financedtos.AdjustmentRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if body.Direction != "credit" && body.Direction != "debit" {
		return echo.NewHTTPError(http.StatusBadRequest, "direction debe ser credit o debit")
	}

	resp, err := h.staffSvc.RecordAdjustment(ctx, shopID, session.UserID, financeModels.EntityTypeShop, body.Direction, body.AmountBs, body.Notes)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func mapShopPayables(resp financedtos.PayablesResponse) shopPayablesResponse {
	items := make([]shopPayableItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, shopPayableItem{
			ShopID:     it.EntityID,
			Name:       it.Name,
			BalanceBs:  it.BalanceBs,
			BalanceUSD: it.BalanceUSD,
			BcvRate:    it.BcvRate,
			UpdatedAt:  it.UpdatedAt,
		})
	}
	return shopPayablesResponse{Items: items, Total: resp.Total}
}
