package handler

import (
	"net/http"
	"strconv"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	financedtos "Encargalo.app-api.go/internal/finance/domain/dtos"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	financePorts "Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StaffWithdrawalHandler interface {
	List(c echo.Context) error
	Claim(c echo.Context) error
	Complete(c echo.Context) error
	Reject(c echo.Context) error
}

type staffWithdrawalHandler struct {
	svc financePorts.WithdrawalApp
}

func NewStaffWithdrawalHandler(svc financePorts.WithdrawalApp) StaffWithdrawalHandler {
	return &staffWithdrawalHandler{svc: svc}
}

// List godoc
//
// @Summary      Solicitudes de retiro de riders (staff)
// @Description  Lista las solicitudes de retiro. Sin parámetro status devuelve las activas (pending y processing), que son las que requieren acción. Cada ítem trae amount_bs (lo solicitado) y payable_bs (lo que hay que transferir hoy): difieren cuando el rider cobró efectivo después de pedir el retiro, y el que manda es payable_bs. payable_frozen indica si ya quedó congelado por un reclamo. El listado es paginado: total es el total que matchea el filtro, no el tamaño de la página.
// @Tags         Riders Staff Finance
// @Produce      json
// @Param        status  query  string  false  "Filtro por estado"  Enums(pending, processing, completed, rejected)
// @Param        page    query  int     false  "Página (default 1)"
// @Param        limit   query  int     false  "Límite por página, máx 50 (default 20)"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.WithdrawalsResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/staff/finance/withdrawals [get]
func (h *staffWithdrawalHandler) List(c echo.Context) error {
	statuses, err := parseWithdrawalStatuses(c.QueryParam("status"))
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}

	resp, err := h.svc.List(c.Request().Context(), financeModels.EntityTypeRider, statuses, page, limit)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// Claim godoc
//
// @Summary      Reclamar una solicitud de retiro (staff)
// @Description  Reclama la solicitud antes de transferir y congela el monto pagable: ese es el valor exacto a transferir. El reclamo es obligatorio porque la transferencia ocurre fuera del sistema, y congelar acá evita que un movimiento del rider durante la transferencia descuadre lo que se registra. Si el monto pagable neto resulta cero o negativo (el efectivo cobrado consumió el retiro), la solicitud se anula y no hay nada que transferir.
// @Tags         Riders Staff Finance
// @Produce      json
// @Param        id  path  string  true  "ID de la solicitud"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.ClaimResponse  "monto congelado a transferir"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Solicitud no encontrada"
// @Failure      409  {string}  string  "La solicitud ya fue reclamada o resuelta"
// @Failure      422  {string}  string  "No hay monto por transferir"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/staff/finance/withdrawals/{id}/claim [post]
func (h *staffWithdrawalHandler) Claim(c echo.Context) error {
	ctx := c.Request().Context()
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.Claim(ctx, requestID, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// Complete godoc
//
// @Summary      Registrar el pago de un retiro (staff)
// @Description  Cierra la solicitud registrando la transferencia. El comprobante es obligatorio. El monto registrado es el congelado al reclamar y no se recalcula: la transferencia ya ocurrió por ese valor. Si lo transferido difiere del congelado, la corrección va por un ajuste de saldo aparte.
// @Tags         Riders Staff Finance
// @Accept       json
// @Produce      json
// @Param        id    path  string                            true  "ID de la solicitud"
// @Param        body  body  financedtos.CompleteWithdrawalRequest   true  "Comprobante de la transferencia"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.WithdrawalItem
// @Failure      400  {string}  string  "Comprobante requerido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Solicitud no encontrada"
// @Failure      409  {string}  string  "La solicitud no fue reclamada o ya está resuelta"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/staff/finance/withdrawals/{id}/complete [post]
func (h *staffWithdrawalHandler) Complete(c echo.Context) error {
	ctx := c.Request().Context()
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	var req financedtos.CompleteWithdrawalRequest
	if err = c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "body inválido")
	}
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.Complete(ctx, requestID, session.UserID, req)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// Reject godoc
//
// @Summary      Rechazar una solicitud de retiro (staff)
// @Description  Rechaza la solicitud devolviendo el saldo retenido al rider. El motivo es obligatorio y queda visible para el rider. Un rechazo no activa el cooldown de 24h: el rider puede volver a solicitar de inmediato, porque no se le pagó nada.
// @Tags         Riders Staff Finance
// @Accept       json
// @Produce      json
// @Param        id    path  string                          true  "ID de la solicitud"
// @Param        body  body  financedtos.RejectWithdrawalRequest   true  "Motivo del rechazo"
// @Security     SessionCookie
// @Success      200  {object}  financedtos.WithdrawalItem
// @Failure      400  {string}  string  "Motivo requerido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Solicitud no encontrada"
// @Failure      409  {string}  string  "La solicitud ya está resuelta"
// @Failure      500  {string}  string  "Error interno"
// @Router       /riders/staff/finance/withdrawals/{id}/reject [post]
func (h *staffWithdrawalHandler) Reject(c echo.Context) error {
	ctx := c.Request().Context()
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	var req financedtos.RejectWithdrawalRequest
	if err = c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "body inválido")
	}
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.Reject(ctx, requestID, session.UserID, req)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// parseWithdrawalStatuses deja el default en las activas: son las que el
// equipo tiene que atender.
func parseWithdrawalStatuses(raw string) ([]financeModels.WithdrawalStatus, error) {
	if raw == "" {
		return []financeModels.WithdrawalStatus{financeModels.WithdrawalPending, financeModels.WithdrawalProcessing}, nil
	}

	status := financeModels.WithdrawalStatus(raw)
	switch status {
	case financeModels.WithdrawalPending, financeModels.WithdrawalProcessing,
		financeModels.WithdrawalCompleted, financeModels.WithdrawalRejected:
		return []financeModels.WithdrawalStatus{status}, nil
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "status inválido")
	}
}
