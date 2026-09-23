package handler

import (
	"encoding/json"
	"log/slog"

	"Encargalo.app-api.go/internal/auth/domain/models"
	paymentsdtos "Encargalo.app-api.go/internal/payments/domain/dtos"
	paymentsmodels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsports "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/ws"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type StaffReviews interface {
	HandleWebSocket(c echo.Context) error
}

type staffReviews struct {
	hub      paymentsports.StaffReviewHub
	reviews  paymentsports.PaymentReviewsApp
	upgrader websocket.Upgrader
}

func NewStaffReviewsHandler(hub paymentsports.StaffReviewHub, reviews paymentsports.PaymentReviewsApp) StaffReviews {
	cfg := config.Get()
	return &staffReviews{
		hub:      hub,
		reviews:  reviews,
		upgrader: ws.BuildUpgrader(cfg.IsProduction(), cfg.Server.AllowedOrigins),
	}
}

// HandleWebSocket godoc
//
// @Summary WebSocket de revisión de comprobantes (Staff CRM)
// @Description
// Establece una conexión WebSocket para que el staff valide comprobantes de pago
// de órdenes y, más adelante, de viajes. Al conectar, envía el snapshot de reviews
// pendientes y `{"type":"connected"}`.
//
// ### Mensajes entrantes
// - approve: `{"type":"approve","review_id":"...","reference_last4":"1234"}`
// - approve_overpay: `{"type":"approve_overpay","review_id":"...","observed_amount_bs":123.45,"reference_last4":"1234"}`
// - reject_underpaid: `{"type":"reject_underpaid","review_id":"...","observed_amount_bs":100,"reference_last4":"1234"}` — rechaza la referencia por pago incompleto y acredita lo pagado (observed_amount_bs) al balance del cliente de inmediato.
// - reject_invalid: `{"type":"reject_invalid","review_id":"...","reason":"..."}` — no requiere reference_last4 (ej. foto que no es un comprobante).
//
// Si `reference_last4` ya existe en otra review consumida (approved/overpaid/rejected_underpaid),
// sin importar el tipo de referencia, el backend auto-rechaza como `rejected_invalid`
// (comprobante duplicado) y responde `action_ok` con `action:"reject_invalid"` y `duplicate_reject:true`.
//
// `action_ok` incluye `reference_id` y `reference_type`. El estado final de la revisión
// llega en el broadcast `payment_review_update`.
//
// @Tags         Payments
// @Security     SessionCookie
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /payments/staff/reviews [get]
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

		var payload paymentsdtos.StaffWSMessage
		if err := json.Unmarshal(msg, &payload); err != nil {
			client.Send([]byte(`{"type":"error","message":"payload inválido"}`))
			continue
		}

		var result paymentsmodels.StaffReviewResult

		switch payload.Type {
		case "approve":
			if payload.ReferenceLast4 == "" {
				client.Send([]byte(`{"type":"error","message":"reference_last4 requerido"}`))
				continue
			}
			result, err = h.reviews.Approve(ctx, payload.ReviewID, session.UserID, payload.ReferenceLast4)
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
		case "reject_invalid":
			result, err = h.reviews.RejectInvalid(ctx, payload.ReviewID, session.UserID, payload.Reason)
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
			"type":           "action_ok",
			"action":         result.Action,
			"review_id":      result.ReviewID,
			"reference_id":   result.ReferenceID,
			"reference_type": result.ReferenceType,
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
