package config

import (
	"log"
	"os"
	"strings"
	"sync"

	"github.com/shepherd-go/gosuite/config"
)

var (
	Once sync.Once
	cfg  *Config
)

func Get() *Config {
	if cfg == nil {
		log.Panic("Configuration has not yet been initialized")
	}
	return cfg
}

type Config struct {
	Server          Server          `env:"server"`
	Postgres        Postgres        `env:"postgres"`
	Redis           Redis           `env:"redis"`
	RedisStream     RedisStream     `env:"redis_stream"`
	JWT             JWT             `env:"jwt"`
	Environment     Environment     `env:"environment"`
	Google          Google          `env:"google"`
	Cloudinary      Cloudinary      `env:"cloudinary"`
	Observability   Observability   `env:"observability"`
	ExternalAPI     ExternalAPI     `env:"external_api"`
	InternalGateway InternalGateway `env:"internal_gateway"`
}

type Server struct {
	Port           int    `validate:"required" env:"port"`
	AllowedOrigins string `env:"allowed_origins"`
}

type Postgres struct {
	Host     string `validate:"required" env:"host"`
	Port     int    `validate:"required" env:"port"`
	User     string `validate:"required" env:"user"`
	Password string `validate:"required" env:"password"`
	Database string `validate:"required" env:"database"`
}

type Redis struct {
	URL string `validate:"required" env:"url"`
}

type RedisStream struct {
	StreamName string `validate:"required" env:"stream_name"`
	Db         int    `validate:"min=0" env:"db"`
}

type JWT struct {
	Secret string `validate:"required" env:"secret"`
}

type Environment struct {
	Current string `validate:"required" env:"current"`
}

// IsProduction is true when ENVIRONMENT_CURRENT is production.
// Railway uses "produccion"; docs and some services use "production".
func (e Environment) IsProduction() bool {
	switch strings.ToLower(strings.TrimSpace(e.Current)) {
	case "production", "produccion", "producción", "prod":
		return true
	default:
		return false
	}
}

func (c *Config) IsProduction() bool {
	if c == nil {
		return false
	}
	return c.Environment.IsProduction()
}

type Google struct {
	ServerKey string `validate:"required" env:"server_key"`
	ClientID  string `validate:"required" env:"client_id"`
}

type Cloudinary struct {
	URL    string `validate:"required" env:"url"`
	Folder string `validate:"required" env:"folder"`
}

// Observability configura el envío de trazas/logs al OTel Collector.
// OtlpEndpoint vacío deshabilita la telemetría (útil en desarrollo local).
// OtlpToken viaja como "Authorization: Bearer <token>" — requerido cuando el
// collector está expuesto públicamente (entorno "observabilidad" separado).
type Observability struct {
	OtlpEndpoint string `env:"otlp_endpoint"`
	OtlpToken    string `env:"otlp_token"`
}

func (o Observability) Enabled() bool {
	return o.OtlpEndpoint != ""
}

// ExternalAPI contiene el API key estático que deben mandar los clientes
// externos vía "Authorization: Bearer <key>" para consumir rutas públicas
// no ligadas a la sesión de usuario (ej. /bcv/rate).
type ExternalAPI struct {
	Key string `validate:"required" env:"key"`
}

// InternalGateway contiene el secreto compartido que el reverse proxy
// (Traefik) inyecta en cada request que reenvía al backend. Lo validan
// AuthMiddleware.Auth y el endpoint /internal/verify-session mediante
// comparación de tiempo constante, como defensa en profundidad además del
// aislamiento de red privada.
type InternalGateway struct {
	Secret string `validate:"required" env:"secret"`
}

func Environments() {
	Once.Do(func() {
		// Railway healthcheck y el private DNS usan PORT; el tag espera SERVER_PORT.
		if os.Getenv("SERVER_PORT") == "" {
			if p := os.Getenv("PORT"); p != "" {
				_ = os.Setenv("SERVER_PORT", p)
			}
		}

		cfg = new(Config)
		if err := config.GetConfigFromEnv(cfg); err != nil {
			log.Panicf("error parsing enviroment vars \n%v", err)
		}
	})
}
