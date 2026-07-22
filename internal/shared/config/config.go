package config

import (
	"log"
	"strings"
	"sync"
	"time"

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
	OlapPostgres  OlapPostgres  `env:"olap_postgres"`
	ETL           ETL           `env:"etl"`
	Redis         Redis         `env:"redis"`
	RedisStream   RedisStream   `env:"redis_stream"`
	JWT           JWT           `env:"jwt"`
	Slack         Slack         `env:"slack"`
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

// OlapPostgres es opcional para la API; el servicio ETL exige Host != "".
type OlapPostgres struct {
	Host     string `env:"host"`
	Port     int    `env:"port"`
	User     string `env:"user"`
	Password string `env:"password"`
	Database string `env:"database"`
}

func (o OlapPostgres) Configured() bool {
	return o.Host != "" && o.Database != "" && o.User != "" && o.Port > 0
}

// ETL configura el worker de sincronización OLAP (solo usado por cmd/etl).
type ETL struct {
	// Schedule es expresión cron (5 campos). Vacío = "48 * * * *" (minuto 48 de cada hora).
	Schedule string `env:"schedule"`
	// LookbackHours: ventana de re-lectura tras watermark (default 48).
	LookbackHours int `env:"lookback_hours"`
	// InitialLookbackDays: backfill si no hay watermark (default 1825 = 5 años).
	InitialLookbackDays int `env:"initial_lookback_days"`
}

func (e ETL) DefaultLookback() time.Duration {
	h := e.LookbackHours
	if h <= 0 {
		h = 48
	}
	return time.Duration(h) * time.Hour
}

func (e ETL) InitialLookback() time.Duration {
	d := e.InitialLookbackDays
	if d <= 0 {
		d = 365 * 5
	}
	return time.Duration(d) * 24 * time.Hour
}

const DefaultETLSchedule = "48 * * * *" // minuto 48 de cada hora

// ResolvedSchedule devuelve el cron efectivo. Vacío tras "off" = desactivado.
func (e ETL) ResolvedSchedule() string {
	s := strings.TrimSpace(e.Schedule)
	switch strings.ToLower(s) {
	case "off", "disabled", "false", "-":
		return ""
	case "":
		return DefaultETLSchedule
	default:
		return s
	}
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

type Slack struct {
	Token   string `env:"token"`
	Channel string `env:"channel"`
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
type Observability struct {
	OtlpEndpoint string `env:"otlp_endpoint"`
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
