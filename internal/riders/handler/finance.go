package handler

import (
	"net/http"
	"strconv"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	_ "Encargalo.app-api.go/internal/riders/domain/dtos"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
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
// @Description  Retorna el balance actual del rider, su zona de control y los últimos 10 movimientos.
// @Tags         Riders Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.BalanceResponse  "balance, zona y últimos 10 movimientos"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/balance [get]
func (h *financeHandler) GetBalance(c echo.Context) error {
	ctx := c.Request().Context()
	session, ok := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if !ok {
		return errcustom.ErrUnautorized
	}

	resp, err := h.svc.GetBalance(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// RequestWithdrawal godoc
//
// @Summary      Solicitar retiro de balance
// @Description  El rider solicita retirar su balance acumulado. Requiere balance >= +$15.
// @Tags         Riders Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.WithdrawalResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      422  {string}  string  "Balance insuficiente para retiro"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/withdrawal [post]
func (h *financeHandler) RequestWithdrawal(c echo.Context) error {
	ctx := c.Request().Context()
	session, ok := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if !ok {
		return errcustom.ErrUnautorized
	}

	resp, err := h.svc.RequestWithdrawal(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// ListTransactions godoc
//
// @Summary      Historial de movimientos del rider
// @Description  Retorna el historial paginado de movimientos de balance del rider.
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
	session, ok := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if !ok {
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

	resp, err := h.svc.ListTransactions(ctx, session.UserID, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}
