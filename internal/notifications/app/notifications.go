package app

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"Encargalo.app-api.go/internal/notifications/domain/models"
	"Encargalo.app-api.go/internal/notifications/notifier"
	"Encargalo.app-api.go/internal/notifications/repo"
	"github.com/google/uuid"
)

type Notifications interface {
	OrderCreated(orderID string) error
	OrderCompleted(orderID string) error
	OrderRejected(orderID string) error
	OrderPendingUnassigned(orderID string, minutesPending int) error
	OrderAutoRejected(orderID string, minutesPending int) error
	OrderNoRider(orderID string) error
	PaymentProofPending(reviewID string, minutesPending int) error
	BalanceRefundPending(requestID string, minutesPending int) error
	WithdrawalRequested(requestID string) error
	TripCreated(tripID string) error
	TripNoDriver(tripID string) error
	GrafanaAlert(payload models.GrafanaWebhookPayload) error
}

type notifications struct {
	repo repo.Repository
	dest notifier.Destination
}

func NewNotifications(repo repo.Repository, dest notifier.Destination) Notifications {
	return &notifications{repo: repo, dest: dest}
}

func (n *notifications) OrderCreated(orderID string) error {
	order, err := n.repo.SearchOrderByID(context.Background(), orderID)
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return err
	}

	msg := models.NotificationMessage{
		Kind:  models.KindOrderCreated,
		Title: "📦 Nueva Orden Recibida",
		Color: 3447003,
		Fields: []models.NotificationField{
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Total", Value: formatUSDAndBs(order.Total, order.BcvRate()), Inline: true},
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Cliente", Value: fmt.Sprintf("%s %s", order.Customer.FirstName, order.Customer.LastName), Inline: true},
		},
		Footer: "Encargalo.app",
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}

	return nil
}

func (n *notifications) OrderCompleted(orderID string) error {
	order, err := n.repo.SearchOrderByID(context.Background(), orderID)
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return err
	}

	msg := models.NotificationMessage{
		Kind:  models.KindOrderCompleted,
		Title: "✅ Orden Completada",
		Color: 3066993,
		Fields: []models.NotificationField{
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Total", Value: formatUSDAndBs(order.Total, order.BcvRate()), Inline: true},
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Cliente", Value: fmt.Sprintf("%s %s", order.Customer.FirstName, order.Customer.LastName), Inline: true},
		},
		Footer: "Encargalo.app",
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}

	return nil
}

func (n *notifications) OrderRejected(orderID string) error {
	order, err := n.repo.SearchOrderByID(context.Background(), orderID)
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return err
	}

	msg := models.NotificationMessage{
		Kind:  models.KindOrderRejected,
		Title: "❌ Orden Rechazada",
		Color: 15158332,
		Fields: []models.NotificationField{
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Total", Value: formatUSDAndBs(order.Total, order.BcvRate()), Inline: true},
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Cliente", Value: fmt.Sprintf("%s %s", order.Customer.FirstName, order.Customer.LastName), Inline: true},
		},
		Footer: "Encargalo.app",
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}

	return nil
}

func (n *notifications) OrderPendingUnassigned(orderID string, minutesPending int) error {
	order, err := n.repo.SearchOrderByID(context.Background(), orderID)
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return err
	}

	address := order.Shop.Address
	if address == "" {
		address = "—"
	}

	waitLabel := formatMinutesPending(minutesPending)

	msg := models.NotificationMessage{
		Kind:  models.KindOrderPendingUnassigned,
		Title: "⚠️ Orden sin respuesta del local",
		Color: 16776960, // yellow
		Fields: []models.NotificationField{
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Tiempo sin aceptar", Value: waitLabel, Inline: true},
			{Name: "Subtotal (sin delivery)", Value: formatUSDAndBs(order.SubtotalWithoutDelivery(), order.BcvRate()), Inline: true},
			{Name: "Owner", Value: order.Shop.Owner.FullName(), Inline: true},
			{Name: "Teléfono Owner", Value: order.Shop.Owner.PhoneOrDash(), Inline: true},
			{Name: "Dirección del local", Value: address, Inline: false},
		},
		Footer: fmt.Sprintf("Encargalo.app · %s sin aceptación", waitLabel),
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}

	return nil
}

func (n *notifications) OrderAutoRejected(orderID string, minutesPending int) error {
	order, err := n.repo.SearchOrderByID(context.Background(), orderID)
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return err
	}

	address := order.Shop.Address
	if address == "" {
		address = "—"
	}
	waitLabel := formatMinutesPending(minutesPending)

	msg := models.NotificationMessage{
		Kind:  models.KindOrderAutoRejected,
		Title: "⏰ Orden cancelada — el local no aceptó",
		Color: 15158332, // red
		Fields: []models.NotificationField{
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Tiempo sin aceptar", Value: waitLabel, Inline: true},
			{Name: "Total", Value: formatUSDAndBs(order.Total, order.BcvRate()), Inline: true},
			{Name: "Cliente", Value: order.Customer.FullName(), Inline: true},
			{Name: "Owner", Value: order.Shop.Owner.FullName(), Inline: true},
			{Name: "Teléfono Owner", Value: order.Shop.Owner.PhoneOrDash(), Inline: true},
			{Name: "Dirección del local", Value: address, Inline: false},
		},
		Footer: fmt.Sprintf("Encargalo.app · auto-cancelada a los %s", waitLabel),
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}
	return nil
}

func (n *notifications) OrderNoRider(orderID string) error {
	order, err := n.repo.SearchOrderByID(context.Background(), orderID)
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return err
	}

	msg := models.NotificationMessage{
		Kind:  models.KindOrderNoRider,
		Title: "🛵 Orden sin repartidor",
		Color: 16776960, // yellow
		Fields: []models.NotificationField{
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Total", Value: formatUSDAndBs(order.Total, order.BcvRate()), Inline: true},
			{Name: "Cliente", Value: order.Customer.FullName(), Inline: true},
			{Name: "Dirección del local", Value: orDash(order.Shop.Address), Inline: false},
		},
		Footer: "Encargalo.app · ningún repartidor aceptó en 10 minutos",
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}
	return nil
}

func (n *notifications) PaymentProofPending(reviewID string, minutesPending int) error {
	review, err := n.repo.SearchPaymentReviewByID(context.Background(), reviewID)
	if err != nil {
		slog.Error("Error searching payment review by ID", "error", err)
		return err
	}
	if review.ID == (uuid.UUID{}) {
		slog.Error("Payment review not found", "reviewID", reviewID)
		return fmt.Errorf("payment review not found: %s", reviewID)
	}

	var msg models.NotificationMessage
	if review.ReferenceType == models.ReferenceTypeTrip {
		msg, err = n.tripPaymentProofMessage(review, minutesPending)
	} else {
		msg, err = n.orderPaymentProofMessage(review, minutesPending)
	}
	if err != nil {
		return err
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}
	return nil
}

func paymentProofTitle(minutesPending int) string {
	if minutesPending > 0 {
		return "🧾 Comprobante pendiente de verificar"
	}
	return "🧾 Nuevo comprobante por verificar"
}

func (n *notifications) orderPaymentProofMessage(review models.PaymentReview, minutesPending int) (models.NotificationMessage, error) {
	order, err := n.repo.SearchOrderByID(context.Background(), review.ReferenceID.String())
	if err != nil {
		slog.Error("Error searching order by ID", "error", err)
		return models.NotificationMessage{}, err
	}

	waitLabel := formatMinutesPending(minutesPending)
	rate := review.BcvRate
	if rate <= 0 {
		rate = order.BcvRate()
	}

	return models.NotificationMessage{
		Kind:  models.KindPaymentProofPending,
		Title: paymentProofTitle(minutesPending),
		URL:   review.ProofURL,
		Color: 3447003, // blue
		Fields: []models.NotificationField{
			{Name: "Local", Value: order.Shop.Name, Inline: true},
			{Name: "Número De Orden", Value: fmt.Sprintf("#%d", order.Number), Inline: true},
			{Name: "Tiempo pendiente", Value: waitLabel, Inline: true},
			{Name: "Monto a verificar", Value: formatBsAndUSD(review.ExpectedAmountBs, rate), Inline: true},
			{Name: "Total orden", Value: formatUSDAndBs(order.Total, rate), Inline: true},
			{Name: "Cliente", Value: order.Customer.FullName(), Inline: true},
		},
		Footer: fmt.Sprintf("Encargalo.app · %s sin verificar", waitLabel),
	}, nil
}

func (n *notifications) tripPaymentProofMessage(review models.PaymentReview, minutesPending int) (models.NotificationMessage, error) {
	trip, err := n.searchTrip(review.ReferenceID.String())
	if err != nil {
		return models.NotificationMessage{}, err
	}

	waitLabel := formatMinutesPending(minutesPending)
	rate := review.BcvRate
	if rate <= 0 {
		rate = trip.Rate()
	}

	return models.NotificationMessage{
		Kind:  models.KindPaymentProofPending,
		Title: paymentProofTitle(minutesPending),
		URL:   review.ProofURL,
		Color: 3447003, // blue
		Fields: []models.NotificationField{
			{Name: "Viaje", Value: trip.ShortID(), Inline: true},
			{Name: "Tiempo pendiente", Value: waitLabel, Inline: true},
			{Name: "Monto a verificar", Value: formatBsAndUSD(review.ExpectedAmountBs, rate), Inline: true},
			{Name: "Tarifa del viaje", Value: formatUSDAndBs(trip.FareUSD(), rate), Inline: true},
			{Name: "Pago", Value: trip.PaymentLabel(), Inline: true},
			{Name: "Cliente", Value: trip.Customer.FullName(), Inline: true},
		},
		Footer: fmt.Sprintf("Encargalo.app · %s sin verificar", waitLabel),
	}, nil
}

func (n *notifications) searchTrip(tripID string) (models.Trip, error) {
	trip, err := n.repo.SearchTripByID(context.Background(), tripID)
	if err != nil {
		slog.Error("Error searching trip by ID", "error", err)
		return models.Trip{}, err
	}
	if trip.ID == (uuid.UUID{}) {
		slog.Error("Trip not found", "tripID", tripID)
		return models.Trip{}, fmt.Errorf("trip not found: %s", tripID)
	}
	return trip, nil
}

func tripFields(trip models.Trip) []models.NotificationField {
	return []models.NotificationField{
		{Name: "Viaje", Value: trip.ShortID(), Inline: true},
		{Name: "Tarifa", Value: formatUSDAndBs(trip.FareUSD(), trip.Rate()), Inline: true},
		{Name: "Pago", Value: trip.PaymentLabel(), Inline: true},
		{Name: "Cliente", Value: trip.Customer.FullName(), Inline: true},
		{Name: "Origen", Value: orDash(trip.PickupAddress), Inline: false},
		{Name: "Destino", Value: orDash(trip.DropoffAddress), Inline: false},
	}
}

func (n *notifications) TripCreated(tripID string) error {
	trip, err := n.searchTrip(tripID)
	if err != nil {
		return err
	}

	msg := models.NotificationMessage{
		Kind:   models.KindTripCreated,
		Title:  "🚗 Nuevo Viaje Solicitado",
		Color:  3447003,
		Fields: tripFields(trip),
		Footer: "Encargalo.app",
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}
	return nil
}

func (n *notifications) TripNoDriver(tripID string) error {
	trip, err := n.searchTrip(tripID)
	if err != nil {
		return err
	}

	waitLabel := formatMinutesPending(int(time.Since(trip.RequestedAt) / time.Minute))
	fields := append(tripFields(trip), models.NotificationField{Name: "Tiempo buscando", Value: waitLabel, Inline: true})

	msg := models.NotificationMessage{
		Kind:   models.KindTripNoDriver,
		Title:  "🚫 Viaje sin conductor",
		Color:  15158332, // red
		Fields: fields,
		Footer: fmt.Sprintf("Encargalo.app · sin conductor tras %s", waitLabel),
	}

	if err = n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func formatMinutesPending(minutes int) string {
	if minutes <= 0 {
		return "recién creada"
	}
	if minutes == 1 {
		return "1 minuto"
	}
	return fmt.Sprintf("%d minutos", minutes)
}

func (n *notifications) BalanceRefundPending(requestID string, minutesPending int) error {
	req, orderStatus, err := n.repo.SearchBalanceRefundRequestByID(context.Background(), requestID)
	if err != nil {
		slog.Error("Error searching balance refund request by ID", "error", err)
		return err
	}
	if req.ID == (uuid.UUID{}) {
		slog.Error("Balance refund request not found", "requestID", requestID)
		return fmt.Errorf("balance refund request not found: %s", requestID)
	}

	customerName := req.Customer.FullName()
	if customerName == "" {
		customerName = "—"
	}
	if orderStatus == "" {
		orderStatus = "—"
	}
	refundStatus := req.Status
	if refundStatus == "" {
		refundStatus = "—"
	}
	waitLabel := formatMinutesPending(minutesPending)

	msg := models.NotificationMessage{
		Kind:  models.KindBalanceRefundPending,
		Title: "💸 Devolución pendiente de completar",
		Color: 15105570, // orange
		Fields: []models.NotificationField{
			{Name: "Cliente", Value: customerName, Inline: true},
			{Name: "Monto", Value: fmt.Sprintf("Bs %.2f", req.AmountBs), Inline: true},
			{Name: "Tiempo pendiente", Value: waitLabel, Inline: true},
			{Name: "Estado solicitud", Value: refundStatus, Inline: true},
			{Name: "Estado de la orden", Value: orderStatus, Inline: true},
		},
		Footer: fmt.Sprintf("Encargalo.app · %s sin completar", waitLabel),
	}

	if err := n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}
	return nil
}

func (n *notifications) GrafanaAlert(payload models.GrafanaWebhookPayload) error {
	if len(payload.Alerts) == 0 {
		return nil
	}

	msgs := make([]models.NotificationMessage, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {
		msgs = append(msgs, buildGrafanaAlertMessage(alert))
	}

	if err := n.dest.Send(context.Background(), msgs); err != nil {
		slog.Error("Error sending Grafana alert notification", "error", err)
		return err
	}

	return nil
}

func buildGrafanaAlertMessage(alert models.GrafanaAlert) models.NotificationMessage {
	msg := models.NotificationMessage{
		Kind:             models.KindGrafanaFiring,
		Title:            "🔴 Error detectado",
		URL:              alert.GeneratorURL,
		Color:            15158332, // red
		AlertEnvironment: alert.Labels["deployment_environment_name"],
		Fields: []models.NotificationField{
			{Name: "Servicio", Value: labelOrDash(alert.Labels, "service_name"), Inline: true},
			{Name: "Entorno", Value: labelOrDash(alert.Labels, "deployment_environment_name"), Inline: true},
			{Name: "Tipo de error", Value: labelOrDash(alert.Labels, "exception_type"), Inline: true},
			{Name: "Mensaje", Value: truncateText(labelOrDash(alert.Labels, "exception_message"), 500), Inline: false},
		},
		Footer: fmt.Sprintf("Grafana · %s", labelOrDash(alert.Labels, "alertname")),
	}
	if alert.Status == "resolved" {
		msg.Kind = models.KindGrafanaResolved
		msg.Title = "✅ Error resuelto"
		msg.Color = 3066993 // green
	}
	return msg
}

func labelOrDash(labels map[string]string, key string) string {
	if value, ok := labels[key]; ok && value != "" {
		return value
	}
	return "—"
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func formatUSDAndBs(usd, bcvRate float64) string {
	if bcvRate > 0 {
		bs := math.Round(usd*bcvRate*100) / 100
		return fmt.Sprintf("Bs %.2f (≈ $%.2f)", bs, usd)
	}
	return fmt.Sprintf("$%.2f", usd)
}

func formatBsAndUSD(bs, bcvRate float64) string {
	if bcvRate > 0 {
		usd := math.Round(bs/bcvRate*100) / 100
		return fmt.Sprintf("Bs %.2f (≈ $%.2f)", bs, usd)
	}
	return fmt.Sprintf("Bs %.2f", bs)
}

// WithdrawalRequested avisa que un rider espera una transferencia. Sin este
// aviso la solicitud depende de que alguien esté mirando el CRM.
func (n *notifications) WithdrawalRequested(requestID string) error {
	req, err := n.repo.SearchWithdrawalRequestByID(context.Background(), requestID)
	if err != nil {
		slog.Error("Error searching withdrawal request by ID", "error", err)
		return err
	}
	if req.ID == (uuid.UUID{}) {
		slog.Error("Withdrawal request not found", "requestID", requestID)
		return fmt.Errorf("withdrawal request not found: %s", requestID)
	}

	riderName := req.Rider.FullName()
	if riderName == "" {
		riderName = "—"
	}
	phone := "—"
	if req.PMPhone != nil && *req.PMPhone != "" {
		phone = *req.PMPhone
	}
	bank := "—"
	if req.PMBankCode != nil && *req.PMBankCode != "" {
		bank = *req.PMBankCode
	}

	msg := models.NotificationMessage{
		Kind:  models.KindWithdrawalRequested,
		Title: "🏍️ Retiro solicitado por un rider",
		Color: 3447003, // blue
		Fields: []models.NotificationField{
			{Name: "Rider", Value: riderName, Inline: true},
			{Name: "Monto solicitado", Value: fmt.Sprintf("Bs %.2f", req.Amount), Inline: true},
			{Name: "Pago Móvil", Value: phone, Inline: true},
			{Name: "Banco", Value: bank, Inline: true},
		},
		Footer: "Encargalo.app · reclamar en el CRM antes de transferir",
	}

	if err := n.dest.Send(context.Background(), []models.NotificationMessage{msg}); err != nil {
		slog.Error("Error sending notification", "error", err)
		return err
	}

	return nil
}
