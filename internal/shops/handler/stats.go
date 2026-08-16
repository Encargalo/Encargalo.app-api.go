package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsPorts "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/labstack/echo/v4"
)

type ShopStatsHandler interface {
	GetSummary(c echo.Context) error
	GetTopProducts(c echo.Context) error
	GetSalesOverTime(c echo.Context) error
	GetCustomerStats(c echo.Context) error
	ExportTransactions(c echo.Context) error
}

type shopStatsHandler struct {
	svc shopsPorts.ShopStatsApp
}

func NewShopStatsHandler(svc shopsPorts.ShopStatsApp) ShopStatsHandler {
	return &shopStatsHandler{svc}
}

// GetSummary godoc
//
// @Summary      Resumen estadístico del local
// @Description  Retorna KPIs del período actual y anterior: total ganado, pérdidas por rechazos, órdenes completadas/rechazadas y tasa de rechazo.
// @Tags         Shops Stats
// @Produce      json
// @Security     SessionCookie
// @Param        period  query  string  false  "Período: day, week, month (default: month)"
// @Success      200  {object}  dtos.StatsSummaryResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/stats/summary [get]
func (h *shopStatsHandler) GetSummary(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	period := dtos.StatsPeriod(c.QueryParam("period"))
	if !validStatsPeriod(period) {
		period = dtos.PeriodMonth
	}

	resp, err := h.svc.GetSummary(ctx, *session.ShopID, period)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// GetTopProducts godoc
//
// @Summary      Productos más vendidos
// @Description  Retorna el ranking de productos más vendidos en el período actual y anterior (solo órdenes Completed).
// @Tags         Shops Stats
// @Produce      json
// @Security     SessionCookie
// @Param        period  query  string  false  "Período: day, week, month (default: month)"
// @Param        limit   query  int     false  "Cantidad de productos (max 20, default 10)"
// @Success      200  {object}  dtos.TopProductsResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/stats/top-products [get]
func (h *shopStatsHandler) GetTopProducts(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	period := dtos.StatsPeriod(c.QueryParam("period"))
	if !validStatsPeriod(period) {
		period = dtos.PeriodMonth
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	resp, err := h.svc.GetTopProducts(ctx, *session.ShopID, period, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// GetSalesOverTime godoc
//
// @Summary      Ventas en el tiempo
// @Description  Retorna la serie temporal de ventas (ganadas y perdidas) del período actual y anterior, con granularidad por hora (day) o por día (week/month).
// @Tags         Shops Stats
// @Produce      json
// @Security     SessionCookie
// @Param        period  query  string  false  "Período: day, week, month (default: month)"
// @Param        cursor  query  string  false  "Cursor de paginación (fecha ISO 8601)"
// @Success      200  {object}  dtos.SalesOverTimeResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/stats/sales-over-time [get]
func (h *shopStatsHandler) GetSalesOverTime(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	period := dtos.StatsPeriod(c.QueryParam("period"))
	if !validStatsPeriod(period) {
		period = dtos.PeriodMonth
	}

	var cursor *time.Time
	if raw := c.QueryParam("cursor"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err == nil {
			cursor = &t
		}
	}

	resp, err := h.svc.GetSalesOverTime(ctx, *session.ShopID, period, cursor)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// GetCustomerStats godoc
//
// @Summary      Clientes nuevos vs recurrentes
// @Description  Retorna la cantidad de clientes nuevos (primera compra Completed histórica dentro del período) y recurrentes (ya habían comprado antes del período) del período actual y anterior.
// @Tags         Shops Stats
// @Produce      json
// @Security     SessionCookie
// @Param        period  query  string  false  "Período: day, week, month (default: month)"
// @Success      200  {object}  dtos.CustomerStatsResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/stats/customers [get]
func (h *shopStatsHandler) GetCustomerStats(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	period := dtos.StatsPeriod(c.QueryParam("period"))
	if !validStatsPeriod(period) {
		period = dtos.PeriodMonth
	}

	resp, err := h.svc.GetCustomerStats(ctx, *session.ShopID, period)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ExportTransactions godoc
//
// @Summary      Exportar movimientos en CSV
// @Description  Descarga el historial completo de movimientos del período como archivo CSV.
// @Tags         Shops Stats
// @Produce      text/csv
// @Security     SessionCookie
// @Param        period  query  string  false  "Período: day, week, month (default: month)"
// @Success      200  {file}    binary  "Archivo CSV"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /business/shops/stats/transactions/export [get]
func (h *shopStatsHandler) ExportTransactions(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if session.ShopID == nil {
		return errcustom.ErrUnautorized
	}

	period := dtos.StatsPeriod(c.QueryParam("period"))
	if !validStatsPeriod(period) {
		period = dtos.PeriodMonth
	}

	txs, err := h.svc.ExportTransactions(ctx, *session.ShopID, period)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	filename := "movimientos_" + string(period) + "_" + time.Now().Format("2006-01-02") + ".csv"
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	_ = w.Write([]string{"fecha", "orden_id", "tipo", "metodo_pago", "monto_usd"})

	for _, tx := range txs {
		orderID := ""
		if tx.OrderID != nil {
			orderID = *tx.OrderID
		}
		method := ""
		if tx.PaymentMethod != nil {
			method = *tx.PaymentMethod
		}
		_ = w.Write([]string{
			tx.CreatedAt,
			orderID,
			tx.MovementType,
			method,
			strconv.FormatFloat(tx.Amount, 'f', 2, 64),
		})
	}
	w.Flush()
	return nil
}

func validStatsPeriod(p dtos.StatsPeriod) bool {
	return p == dtos.PeriodDay || p == dtos.PeriodWeek || p == dtos.PeriodMonth
}
