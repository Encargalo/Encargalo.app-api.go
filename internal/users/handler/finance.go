package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/ws"
	"Encargalo.app-api.go/internal/users/domain/dto"
	_ "Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type CustomerFinanceHandler interface {
	GetBalance(c echo.Context) error
	GetPagoMovil(c echo.Context) error
	UpsertPagoMovil(c echo.Context) error
	CreateRefundRequest(c echo.Context) error
	ListRefundRequests(c echo.Context) error
	GetRefundRequest(c echo.Context) error
}

type customerFinanceHandler struct {
	svc ports.CustomerFinanceApp
}

func NewCustomerFinanceHandler(svc ports.CustomerFinanceApp) CustomerFinanceHandler {
	return &customerFinanceHandler{svc}
}

// GetBalance godoc
// @Summary      Balance del cliente
// @Description  Retorna el balance en bolívares (Bs) del cliente y los últimos 10 movimientos. Créditos elegibles incluyen can_request_refund (ventana/monto/saldo; no exige Pago Móvil) y refund_eligible_until. Los débitos de devolución (balance_refund_hold / processing / payout) y créditos balance_refund_restore incluyen refund_request_id para abrir el detalle.
// @Tags         Users Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dto.CustomerBalanceResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /users/balance [get]
func (h *customerFinanceHandler) GetBalance(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.GetBalance(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetPagoMovil godoc
// @Summary      Pago Móvil del cliente
// @Description  Retorna el Pago Móvil registrado para devoluciones.
// @Tags         Users Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dto.PagoMovilResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No registrado"
// @Router       /users/pago-movil [get]
func (h *customerFinanceHandler) GetPagoMovil(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	resp, err := h.svc.GetPagoMovil(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// UpsertPagoMovil godoc
// @Summary      Crear o actualizar Pago Móvil
// @Description  Registra o modifica el Pago Móvil destino del cliente para devoluciones.
// @Tags         Users Finance
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dto.UpsertPagoMovilRequest  true  "Datos Pago Móvil"
// @Success      200  {object}  dto.PagoMovilResponse
// @Failure      400  {string}  string  "Datos inválidos"
// @Failure      401  {string}  string  "No autorizado"
// @Router       /users/pago-movil [put]
func (h *customerFinanceHandler) UpsertPagoMovil(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	var req dto.UpsertPagoMovilRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	resp, err := h.svc.UpsertPagoMovil(ctx, session.UserID, req)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// CreateRefundRequest godoc
// @Summary      Solicitar devolución de crédito
// @Description  Solicita devolución en efectivo de la porción Pago Móvil de un crédito elegible (> $1, ventana 15 min). Requiere Pago Móvil registrado.
// @Tags         Users Finance
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dto.CreateRefundRequest  true  "transaction_id del crédito"
// @Success      201  {object}  dto.RefundRequestResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      409  {string}  string  "Ya existe solicitud"
// @Failure      422  {string}  string  "No elegible / sin PM"
// @Router       /users/balance/refund-requests [post]
func (h *customerFinanceHandler) CreateRefundRequest(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	var req dto.CreateRefundRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	txID, err := uuid.Parse(strings.TrimSpace(req.TransactionID))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "transaction_id inválido")
	}
	resp, err := h.svc.CreateRefundRequest(ctx, session.UserID, txID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// ListRefundRequests godoc
// @Summary      Listar solicitudes de devolución
// @Description  Lista las solicitudes de devolución del cliente autenticado.
// @Tags         Users Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {array}  dto.RefundRequestResponse
// @Failure      401  {string}  string  "No autorizado"
// @Router       /users/balance/refund-requests [get]
func (h *customerFinanceHandler) ListRefundRequests(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	resp, err := h.svc.ListRefundRequests(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// GetRefundRequest godoc
// @Summary      Detalle de solicitud de devolución
// @Description  Detalle de una solicitud (incluye proof_url cuando status=completed). Usar desde el historial con refund_request_id.
// @Tags         Users Finance
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "UUID de la solicitud"
// @Success      200  {object}  dto.RefundRequestResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "No encontrada"
// @Router       /users/balance/refund-requests/{id} [get]
func (h *customerFinanceHandler) GetRefundRequest(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}
	resp, err := h.svc.GetRefundRequest(ctx, session.UserID, id)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

type StaffRefundHandler interface {
	HandleWebSocket(c echo.Context) error
	CompleteRefund(c echo.Context) error
}

type staffRefundHandler struct {
	hub      ports.StaffRefundHub
	finance  ports.CustomerFinanceApp
	upgrader websocket.Upgrader
}

func NewStaffRefundHandler(hub ports.StaffRefundHub, finance ports.CustomerFinanceApp) StaffRefundHandler {
	cfg := config.Get()
	return &staffRefundHandler{
		hub:      hub,
		finance:  finance,
		upgrader: ws.BuildUpgrader(cfg.Environment.Current == "production", cfg.Server.AllowedOrigins),
	}
}

type staffRefundWSMessage struct {
	Type       string    `json:"type"`
	RequestID  uuid.UUID `json:"request_id"`
	ProofImage string    `json:"proof_image,omitempty"`
	AmountBs   float64   `json:"amount_bs,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

// HandleWebSocket godoc
//
// @Summary WebSocket de devoluciones de balance (Staff CRM)
// @Description
// Conexión WebSocket para que el staff gestione solicitudes de devolución.
// Al conectar envía snapshot de pending/accepted y `{"type":"connected"}`.
//
// ### Mensajes entrantes
// - accept: `{"type":"accept","request_id":"..."}` — pending→accepted (el balance ya se debitó al crear)
// - reject: `{"type":"reject","request_id":"...","reason":"..."}` — pending→rejected y restaura el monto al balance
// - complete: `{"type":"complete","request_id":"...","amount_bs":52,"proof_image":"data:image/...;base64,..."}` — solo desde accepted; amount_bs debe coincidir; la imagen se sube a Cloudinary
//
// @Tags         Users Finance
// @Security     SessionCookie
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /users/staff/refund-requests [get]
func (h *staffRefundHandler) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading staff refund websocket", "error", err)
		return err
	}

	client := ws.NewClientWithReadLimit(conn, 8<<20) // 8 MiB para proof_image base64
	go client.WritePump()

	if err := h.hub.Register(ctx, session.UserID, client); err != nil {
		_ = conn.Close()
		return err
	}
	defer h.hub.Unregister(session.UserID, client)

	client.Send([]byte(`{"type":"connected"}`))

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("staff refund ws closed unexpectedly", "error", err)
			}
			break
		}

		var payload staffRefundWSMessage
		if err := json.Unmarshal(msg, &payload); err != nil {
			client.Send([]byte(`{"type":"error","message":"payload inválido"}`))
			continue
		}

		var (
			result dto.RefundRequestResponse
			action string
		)

		switch payload.Type {
		case "accept":
			action = "accept"
			result, err = h.finance.AcceptRefund(ctx, payload.RequestID, session.UserID)
		case "reject":
			action = "reject"
			result, err = h.finance.RejectRefund(ctx, payload.RequestID, session.UserID, payload.Reason)
		case "complete":
			action = "complete"
			result, err = h.finance.CompleteRefund(ctx, payload.RequestID, payload.ProofImage, payload.AmountBs)
		default:
			client.Send([]byte(`{"type":"error","message":"tipo de mensaje desconocido"}`))
			continue
		}

		if err != nil {
			slog.Warn("staff refund action failed", "type", payload.Type, "error", err)
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
			client.Send(errMsg)
			continue
		}

		okMsg, _ := json.Marshal(map[string]any{
			"type":       "action_ok",
			"action":     action,
			"request_id": payload.RequestID,
			"status":     result.Status,
		})
		client.Send(okMsg)
	}

	return nil
}

// CompleteRefund godoc
//
// @Summary      Completar devolución (staff)
// @Description  Registra el pago de una solicitud accepted: valida amount_bs, sube proof_image (base64) a Cloudinary y marca completed.
// @Tags         Users Finance
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path  string  true  "ID de la solicitud"
// @Param        body  body  dto.CompleteRefundRequest  true  "Monto pagado + imagen del comprobante"
// @Success      200  {object}  dto.RefundRequestResponse
// @Failure      400  {string}  string  "Datos inválidos / falta comprobante"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      422  {string}  string  "Monto no coincide / estado inválido"
// @Router       /users/staff/refund-requests/{id}/complete [post]
func (h *staffRefundHandler) CompleteRefund(c echo.Context) error {
	ctx := c.Request().Context()
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	var body dto.CompleteRefundRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if strings.TrimSpace(body.ProofImage) == "" {
		return errcustom.ToHTTPError(errcustom.ErrRefundProofRequired)
	}
	if body.AmountBs <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "amount_bs inválido")
	}

	resp, err := h.finance.CompleteRefund(ctx, requestID, body.ProofImage, body.AmountBs)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}
