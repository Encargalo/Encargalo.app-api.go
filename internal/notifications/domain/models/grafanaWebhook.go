package models

// Payload del webhook de contact point de Grafana Alerting.
type GrafanaWebhookPayload struct {
	Receiver string         `json:"receiver"`
	Status   string         `json:"status"`
	Alerts   []GrafanaAlert `json:"alerts"`
}

type GrafanaAlert struct {
	Status       string            `json:"status"` // firing | resolved
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
	ValueString  string            `json:"valueString"`
}
