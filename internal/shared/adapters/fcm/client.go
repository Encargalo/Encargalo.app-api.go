// Package fcm es un cliente mínimo de FCM HTTP v1 (push a Android/iOS), en el
// mismo estilo que shared/adapters/googlemaps: net/http directo en vez de un
// SDK pesado. golang.org/x/oauth2/google ya es dependencia transitiva del
// proyecto (google.golang.org/api), así que no agrega un módulo nuevo.
package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// ErrInvalidToken señala que FCM respondió que el token ya no está
// registrado (UNREGISTERED/NOT_FOUND) — el llamador debe borrarlo.
var ErrInvalidToken = errors.New("fcm: token no registrado")

type Client struct {
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	projectID   string
	baseURL     string
}

// serviceAccountKey lee solo el campo que JWTConfigFromJSON no expone — el
// resto (private_key, client_email, token_uri) lo valida esa función.
type serviceAccountKey struct {
	ProjectID string `json:"project_id"`
}

// NewClient devuelve (nil, nil) si no hay credenciales configuradas — el
// push es un canal best-effort, nunca debe impedir que el servidor arranque.
// El llamador (y pushFCMNotifier) tratan un *Client nil como "push deshabilitado".
func NewClient(cfg config.Config) (*Client, error) {
	raw := []byte(cfg.Firebase.CredentialsJSON)
	if len(raw) == 0 {
		return nil, nil
	}

	// JWTConfigFromJSON, a diferencia de CredentialsFromJSON*, solo acepta el
	// formato de clave de cuenta de servicio (private_key + client_email +
	// token_uri fijo de Google) — no el formato genérico de "credential
	// configuration" (external_account, impersonación, etc.) que motiva esa
	// deprecación por riesgo de SSRF con una fuente no controlada.
	jwtConfig, err := google.JWTConfigFromJSON(raw, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("fcm: credenciales inválidas: %w", err)
	}

	var key serviceAccountKey
	if err := json.Unmarshal(raw, &key); err != nil || key.ProjectID == "" {
		return nil, fmt.Errorf("fcm: credenciales sin project_id: %w", err)
	}

	return &Client{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		tokenSource: jwtConfig.TokenSource(context.Background()),
		projectID:   key.ProjectID,
		baseURL:     "https://fcm.googleapis.com/v1",
	}, nil
}

type message struct {
	Token        string            `json:"token"`
	Notification notification      `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type sendRequest struct {
	Message message `json:"message"`
}

type fcmErrorBody struct {
	Error struct {
		Status string `json:"status"`
	} `json:"error"`
}

// Send manda un push a un solo token. Devuelve ErrInvalidToken si FCM dice
// que el token ya no existe — el llamador decide si lo borra.
func (c *Client) Send(ctx context.Context, token, title, body string, data map[string]string) error {
	payload := sendRequest{Message: message{Token: token, Notification: notification{Title: title, Body: body}, Data: data}}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("fcm: error serializando mensaje: %w", err)
	}

	url := fmt.Sprintf("%s/projects/%s/messages:send", c.baseURL, c.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	authToken, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("fcm: error obteniendo access token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: error de red: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	var fcmErr fcmErrorBody
	_ = json.Unmarshal(respBody, &fcmErr)
	if fcmErr.Error.Status == "UNREGISTERED" || fcmErr.Error.Status == "NOT_FOUND" {
		return ErrInvalidToken
	}
	return fmt.Errorf("fcm: respuesta %d: %s", resp.StatusCode, string(respBody))
}
