package handler

import (
	"encoding/json"
	"log/slog"

	"Encargalo.app-api.go/internal/auth/domain/models"
	modelsOrder "Encargalo.app-api.go/internal/orders/domain/models"
	portsOrder "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type StaffReviews interface {
	HandleWebSocket(c echo.Context) error
}

type staffReviews struct {
	hub      portsOrder.StaffReviewHub
	reviews  portsOrder.PaymentReviewsApp
	upgrader websocket.Upgrader
}

func NewStaffReviewsHandler(hub portsOrder.StaffReviewHub, reviews portsOrder.PaymentReviewsApp) StaffReviews {
	cfg := config.Get()
	return &staffReviews{
		hub:      hub,
		reviews:  reviews,
		upgrader: ws.BuildUpgrader(cfg.Environment.Current == "production", cfg.Server.AllowedOrigins),
	}
}

type staffWSMessage struct {
	Type             string    `json:"type"`
	ReviewID         uuid.UUID `json:"review_id"`
	ObservedAmountBs *float64  `json:"observed_amount_bs,omitempty"`
	ReferenceLast4   string    `json:"reference_last4,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

// HandleWebSocket godoc
//
// @Summary WebSocket de revisión de comprobantes (Staff CRM)
// @Description
// Establece una conexión WebSocket para que el staff valide comprobantes de pago.
// Al conectar, envía snapshot de reviews pendientes y `{"type":"connected"}`.
//
// ### Mensajes entrantes
// - approve: `{"type":"approve","review_id":"...","reference_last4":"1234"}`
// - approve_overpay: `{"type":"approve_overpay","review_id":"...","observed_amount_bs":123.45,"reference_last4":"1234"}`
// - reject_underpaid: `{"type":"reject_underpaid","review_id":"...","observed_amount_bs":100,"reference_last4":"1234"}` — rechaza la orden por pago incompleto y acredita lo pagado (observed_amount_bs) al balance del cliente de inmediato.
// - reject_invalid: `{"type":"reject_invalid","review_id":"...","reason":"..."}` — no requiere reference_last4 (ej. foto que no es un comprobante).
//
// Si `reference_last4` ya existe en otra review consumida (approved/overpaid/rejected_underpaid),
// el backend auto-rechaza como `rejected_invalid` (comprobante duplicado) y responde
// `action_ok` con `action:"reject_invalid"` y `duplicate_reject:true`.
//
// @Tags         Orders WebSocket
// @Security     SessionCookie
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /orders/staff/reviews [get]
func (h *staffReviews) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading staff websocket", "error", err)
		return err
	}

	client := ws.NewClient(conn)
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
				slog.Warn("staff ws closed unexpectedly", "error", err)
			}
			break
		}

		var payload staffWSMessage
		if err := json.Unmarshal(msg, &payload); err != nil {
			client.Send([]byte(`{"type":"error","message":"payload inválido"}`))
			continue
		}

		var (
			result modelsOrder.StaffReviewResult
			order  modelsOrder.Order
		)

		switch payload.Type {
		case "approve":
			if payload.ReferenceLast4 == "" {
				client.Send([]byte(`{"type":"error","message":"reference_last4 requerido"}`))
				continue
			}
			result, err = h.reviews.Approve(ctx, payload.ReviewID, session.UserID, payload.ReferenceLast4)
			order = result.Order
		case "approve_overpay":
			if payload.ObservedAmountBs == nil {
				client.Send([]byte(`{"type":"error","message":"observed_amount_bs requerido"}`))
				continue
			}
			if payload.ReferenceLast4 == "" {
				client.Send([]byte(`{"type":"error","message":"reference_last4 requerido"}`))
				continue
			}
			result, err = h.reviews.ApproveOverpay(ctx, payload.ReviewID, session.UserID, *payload.ObservedAmountBs, payload.ReferenceLast4)
			order = result.Order
		case "reject_underpaid":
			if payload.ObservedAmountBs == nil {
				client.Send([]byte(`{"type":"error","message":"observed_amount_bs requerido"}`))
				continue
			}
			if payload.ReferenceLast4 == "" {
				client.Send([]byte(`{"type":"error","message":"reference_last4 requerido"}`))
				continue
			}
			result, err = h.reviews.RejectUnderpaid(ctx, payload.ReviewID, session.UserID, *payload.ObservedAmountBs, payload.ReferenceLast4)
			order = result.Order
		case "reject_invalid":
			order, err = h.reviews.RejectInvalid(ctx, payload.ReviewID, session.UserID, payload.Reason)
			result = modelsOrder.StaffReviewResult{Order: order, Action: "reject_invalid"}
		default:
			client.Send([]byte(`{"type":"error","message":"tipo de mensaje desconocido"}`))
			continue
		}

		if err != nil {
			slog.Warn("staff review action failed", "type", payload.Type, "error", err)
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
			client.Send(errMsg)
			continue
		}

		okPayload := map[string]any{
			"type":         "action_ok",
			"action":       result.Action,
			"review_id":    payload.ReviewID,
			"order_id":     order.ID,
			"order_status": order.Status,
		}
		if result.DuplicateReject {
			okPayload["duplicate_reject"] = true
			okPayload["rejection_reason"] = result.RejectionReason
		}
		okMsg, _ := json.Marshal(okPayload)
		client.Send(okMsg)
	}

	return nil
}
