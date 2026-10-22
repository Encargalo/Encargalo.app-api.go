package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/notifications/app"
	"Encargalo.app-api.go/internal/notifications/domain/models"
	"Encargalo.app-api.go/internal/shared/config"
	"golang.org/x/time/rate"
)

type GrafanaWebhook interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

type grafanaWebhook struct {
	svc     app.Notifications
	secret  string
	limiter *rate.Limiter
}

func NewGrafanaWebhook(svc app.Notifications, config config.NotificationsConfig) GrafanaWebhook {
	return &grafanaWebhook{
		svc:    svc,
		secret: config.Grafana.WebhookSecret,
		// 1 req/s sostenido con ráfagas de hasta 5: de sobra para alertas reales de Grafana,
		// pero limita el daño si el secreto se filtra.
		limiter: rate.NewLimiter(rate.Limit(1), 5),
	}
}

func (g *grafanaWebhook) Handle(w http.ResponseWriter, r *http.Request) {
	if !g.limiter.Allow() {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	received := r.Header.Get("X-Webhook-Secret")
	if subtle.ConstantTimeCompare([]byte(received), []byte(g.secret)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var payload models.GrafanaWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("error decodificando payload de webhook de Grafana", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Se procesa en segundo plano para responder rápido y evitar que Grafana reintente por timeout.
	go func() {
		if err := g.svc.GrafanaAlert(payload); err != nil {
			slog.Error("error procesando alerta de Grafana", "error", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
