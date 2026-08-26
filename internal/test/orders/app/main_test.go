package app_test

import (
	"os"
	"testing"

	"Encargalo.app-api.go/internal/shared/config"
)

// TestMain inicializa config.Get() con env vars dummy antes de correr los
// tests del paquete: CreateOrder/CreateQuote llaman config.Get().IsProduction()
// (vía priceOrderDraft) para decidir si usan deliveryfee.Calculate (Google Maps
// real) o deliveryfee.DevPlaceholderFee. No seteamos ENVIRONMENT_CURRENT a
// production, así que IsProduction() da false y los tests nunca llaman a Maps.
func TestMain(m *testing.M) {
	envs := map[string]string{
		"SERVER_PORT":              "3000",
		"POSTGRES_HOST":            "localhost",
		"POSTGRES_PORT":            "5432",
		"POSTGRES_USER":            "test",
		"POSTGRES_PASSWORD":        "test",
		"POSTGRES_DATABASE":        "test",
		"REDIS_URL":                "redis://localhost:6379",
		"REDIS_STREAM_STREAM_NAME": "test-stream",
		"JWT_SECRET":               "test-secret",
		"ENVIRONMENT_CURRENT":      "test",
		"GOOGLE_SERVER_KEY":        "test",
		"GOOGLE_CLIENT_ID":         "test",
		"CLOUDINARY_URL":           "cloudinary://test",
		"CLOUDINARY_FOLDER":        "test",
		"EXTERNAL_API_KEY":         "test",
		"INTERNAL_GATEWAY_SECRET":  "test",
	}
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	config.Environments()

	os.Exit(m.Run())
}
