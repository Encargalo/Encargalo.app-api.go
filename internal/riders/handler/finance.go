package handler

import (
	"net/http"
	"strconv"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	_ "Encargalo.app-api.go/internal/riders/domain/dtos"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
)

type FinanceHandler interface {
	GetBalance(c echo.Context) error
	RequestWithdrawal(c echo.Context) error
	ListTransactions(c echo.Context) error
}

type financeHandler struct {
	svc ridersPorts.FinanceApp
}

func NewFinanceHandler(svc ridersPorts.FinanceApp) FinanceHandler {
	return &financeHandler{svc}
}

// GetBalance godoc
//
// @Summary      Balance del rider
// @Description  Retorna balance_bs y balance_usd (tasa BCV vigente en bcv_rate), zona de control, mínimo de retiro (withdrawal_min_bs), la solicitud de retiro en curso si existe (active_withdrawal, que distingue un saldo en 0 por retiro pendiente de uno por no haber ganado nada) y los últimos 10 movimientos (amount_bs y amount_usd).
// @Tags         Riders Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.BalanceResponse  "balance, zona y últimos 10 movimientos"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/balance [get]
func (h *financeHandler) GetBalance(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.GetBalance(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return response.Send(c, http.StatusOK, resp)
}

// RequestWithdrawal godoc
//
// @Summary      Solicitar retiro de balance
// @Description  Crea una solicitud de retiro, no un pago: retiene el saldo y queda pendiente hasta que staff transfiera y cargue el comprobante. Requiere balance >= al equivalente dinámico de $3 según la tasa BCV vigente (ver withdrawal_min_bs en GET /riders/balance), Pago Móvil registrado, no tener otra solicitud en curso, y que hayan pasado 24h desde el último retiro pagado.
// @Tags         Riders Finance
// @Produce      json
// @Security     SessionCookie
// @Success      201  {object}  dtos.WithdrawalResponse  "solicitud creada"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      409  {string}  string  "Ya existe una solicitud de retiro en curso"
// @Failure      422  {string}  string  "Balance insuficiente para retiro o Pago Móvil no registrado"
// @Failure      429  {string}  string  "Debe esperar 24h desde el último retiro pagado"
// @Failure      503  {string}  string  "Tasa BCV no disponible"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/withdrawal [post]
func (h *financeHandler) RequestWithdrawal(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.RequestWithdrawal(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return response.Send(c, http.StatusCreated, resp)
}

// ListTransactions godoc
//
// @Summary      Historial de movimientos del rider
// @Description  Retorna el historial paginado de movimientos (amount_bs y amount_usd).
// @Tags         Riders Finance
// @Produce      json
// @Param        page   query  int  false  "Página (default 1)"
// @Param        limit  query  int  false  "Límite por página, máx 50 (default 20)"
// @Security     SessionCookie
// @Success      200  {object}  dtos.TransactionsResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/transactions [get]
func (h *financeHandler) ListTransactions(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}

	resp, err := h.svc.ListTransactions(ctx, session.UserID, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return response.Send(c, http.StatusOK, resp)
}
