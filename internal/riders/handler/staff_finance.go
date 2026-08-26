package handler

import (
	"net/http"
	"strconv"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	financedtos "Encargalo.app-api.go/internal/finance/domain/dtos"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	financePorts "Encargalo.app-api.go/internal/finance/domain/ports"
	_ "Encargalo.app-api.go/internal/riders/domain/dtos"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffFinanceHandler interface {
	ListPayables(c echo.Context) error
	GetBalance(c echo.Context) error
	ListTransactions(c echo.Context) error
	RecordPayout(c echo.Context) error
	RecordAdjustment(c echo.Context) error
}

type staffFinanceHandler struct {
	staffSvc financePorts.StaffFinanceApp
	riderSvc ridersPorts.FinanceApp
}

func NewStaffFinanceHandler(staffSvc financePorts.StaffFinanceApp, riderSvc ridersPorts.FinanceApp) StaffFinanceHandler {
	return &staffFinanceHandler{staffSvc: staffSvc, riderSvc: riderSvc}
}

type riderPayablesResponse struct {
	Items []riderPayableItem `json:"items"`
	Total int                `json:"total"`
}

type riderPayableItem struct {
	RiderID    string  `json:"rider_id"`
	Name       string  `json:"name"`
	SurName    string  `json:"sur_name,omitempty"`
	Phone      string  `json:"phone,omitempty"`
	BalanceBs  float64 `json:"balance_bs"`
	BalanceUSD float64 `json:"balance_usd"`
	BcvRate    float64 `json:"bcv_rate,omitempty"`
	UpdatedAt  string  `json:"updated_at"`
}

// ListPayables godoc
//
// @Summary      Riders con saldo por pagar (staff)
// @Description  Lista riders con balance > 0 en balance_bs y balance_usd (tasa BCV vigente).
// @Tags         Riders Staff Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  riderPayablesResponse
// @Failure      401  {string}  string
// @Failure      500  {string}  string
// @Router       /riders/staff/finance/payables [get]
func (h *staffFinanceHandler) ListPayables(c echo.Context) error {
	resp, err := h.staffSvc.ListPayables(c.Request().Context(), financeModels.EntityTypeRider)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, mapRiderPayables(resp))
}

// GetBalance godoc
//
// @Summary      Balance de un rider (staff)
// @Description  Retorna balance_bs, balance_usd y los últimos 10 movimientos del rider.
// @Tags         Riders Staff Finance
// @Produce      json
// @Param        riderId  path  string  true  "ID del rider"
// @Security     SessionCookie
// @Success      200  {object}  dtos.BalanceResponse
// @Failure      401  {string}  string
// @Failure      500  {string}  string
// @Router       /riders/staff/finance/{riderId}/balance [get]
func (h *staffFinanceHandler) GetBalance(c echo.Context) error {
	riderID, err := uuid.Parse(c.Param("riderId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "riderId inválido")
	}
	resp, err := h.riderSvc.GetBalance(c.Request().Context(), riderID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// ListTransactions godoc
//
// @Summary      Movimientos de un rider (staff)
// @Description  Historial paginado de movimientos del rider (amount_bs y amount_usd).
// @Tags         Riders Staff Finance
// @Produce      json
// @Param        riderId  path   string  true   "ID del rider"
// @Param        page     query  int     false  "Página (default 1)"
// @Param        limit    query  int     false  "Límite por página, máx 50 (default 20)"
// @Security     SessionCookie
// @Success      200  {object}  dtos.TransactionsResponse
// @Failure      401  {string}  string
// @Failure      500  {string}  string
// @Router       /riders/staff/finance/{riderId}/transactions [get]
func (h *staffFinanceHandler) ListTransactions(c echo.Context) error {
	riderID, err := uuid.Parse(c.Param("riderId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "riderId inválido")
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}
	resp, err := h.riderSvc.ListTransactions(c.Request().Context(), riderID, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// RecordPayout godoc
//
// @Summary      Registrar pago a rider (staff)
// @Description  Registra un pago manual en Bs (proof_image en base64/data-URL). Debita el balance del rider.
// @Tags         Riders Staff Finance
// @Accept       json
// @Produce      json
// @Param        riderId  path  string                   true  "ID del rider"
// @Param        body     body  financedtos.PayoutRequest  true  "Monto en Bs + comprobante"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.ActionResponse
// @Failure      400  {string}  string
// @Failure      401  {string}  string
// @Failure      422  {string}  string
// @Failure      503  {string}  string
// @Router       /riders/staff/finance/{riderId}/payouts [post]
func (h *staffFinanceHandler) RecordPayout(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	riderID, err := uuid.Parse(c.Param("riderId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "riderId inválido")
	}
	var body financedtos.PayoutRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}

	resp, err := h.staffSvc.RecordPayout(ctx, riderID, session.UserID, financeModels.EntityTypeRider, body.AmountBs, body.Notes, body.ProofImage)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// RecordAdjustment godoc
//
// @Summary      Ajuste manual de balance de rider (staff)
// @Description  Suma o resta saldo en Bs (direction credit/debit). Requiere notes.
// @Tags         Riders Staff Finance
// @Accept       json
// @Produce      json
// @Param        riderId  path  string                       true  "ID del rider"
// @Param        body     body  financedtos.AdjustmentRequest  true  "Dirección, monto en Bs y motivo"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.ActionResponse
// @Failure      400  {string}  string
// @Failure      401  {string}  string
// @Failure      422  {string}  string
// @Failure      503  {string}  string
// @Router       /riders/staff/finance/{riderId}/adjustments [post]
func (h *staffFinanceHandler) RecordAdjustment(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	riderID, err := uuid.Parse(c.Param("riderId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "riderId inválido")
	}
	var body financedtos.AdjustmentRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if body.Direction != "credit" && body.Direction != "debit" {
		return echo.NewHTTPError(http.StatusBadRequest, "direction debe ser credit o debit")
	}

	resp, err := h.staffSvc.RecordAdjustment(ctx, riderID, session.UserID, financeModels.EntityTypeRider, body.Direction, body.AmountBs, body.Notes)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

func mapRiderPayables(resp financedtos.PayablesResponse) riderPayablesResponse {
	items := make([]riderPayableItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, riderPayableItem{
			RiderID:    it.EntityID,
			Name:       it.Name,
			SurName:    it.SurName,
			Phone:      it.Phone,
			BalanceBs:  it.BalanceBs,
			BalanceUSD: it.BalanceUSD,
			BcvRate:    it.BcvRate,
			UpdatedAt:  it.UpdatedAt,
		})
	}
	return riderPayablesResponse{Items: items, Total: resp.Total}
}
