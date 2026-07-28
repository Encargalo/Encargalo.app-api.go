package config

import (
	"log"
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
	Server        Server        `env:"server"`
	Postgres      Postgres      `env:"postgres"`
	Redis         Redis         `env:"redis"`
	RedisStream   RedisStream   `env:"redis_stream"`
	JWT           JWT           `env:"jwt"`
	Environment   Environment   `env:"environment"`
	Google        Google        `env:"google"`
	Cloudinary    Cloudinary    `env:"cloudinary"`
	Observability Observability `env:"observability"`
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

type Google struct {
	Matrix   string `validate:"required" env:"matrix"`
	ClientID string `validate:"required" env:"client_id"`
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

func Environments() {
	Once.Do(func() {
		cfg = new(Config)
		if err := config.GetConfigFromEnv(cfg); err != nil {
			log.Panicf("error parsing enviroment vars \n%v", err)
		}
	})
}
