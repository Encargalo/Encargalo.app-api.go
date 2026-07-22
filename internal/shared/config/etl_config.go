package config

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	gosuite "github.com/shepherd-go/gosuite/config"
)

var (
	etlOnce sync.Once
	etlCfg  *ETLConfig
)

// ETLConfig es la configuración mínima del worker cmd/etl (sin JWT/Redis/Google/Cloudinary).
type ETLConfig struct {
	Server       Server       `env:"server"`
	Postgres     Postgres     `env:"postgres"`
	OlapPostgres OlapPostgres `env:"olap_postgres"`
	ETL          ETLSettings  `env:"etl"`
	Environment  Environment  `env:"environment"`
}

// OlapPostgres destino del warehouse analytics.
type OlapPostgres struct {
	Host     string `validate:"required" env:"host"`
	Port     int    `validate:"required" env:"port"`
	User     string `validate:"required" env:"user"`
	Password string `validate:"required" env:"password"`
	Database string `validate:"required" env:"database"`
}

// ETLSettings cron y ventanas de sync.
type ETLSettings struct {
	Schedule            string `env:"schedule"`
	LookbackHours       int    `env:"lookback_hours"`
	InitialLookbackDays int    `env:"initial_lookback_days"`
}

const DefaultETLSchedule = "48 * * * *"

func (e ETLSettings) DefaultLookback() time.Duration {
	h := e.LookbackHours
	if h <= 0 {
		h = 48
	}
	return time.Duration(h) * time.Hour
}

func (e ETLSettings) InitialLookback() time.Duration {
	d := e.InitialLookbackDays
	if d <= 0 {
		d = 365 * 5
	}
	return time.Duration(d) * 24 * time.Hour
}

func (e ETLSettings) ResolvedSchedule() string {
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

func GetETL() *ETLConfig {
	if etlCfg == nil {
		log.Panic("ETL configuration has not yet been initialized")
	}
	return etlCfg
}

func ETLEnvironments() {
	etlOnce.Do(func() {
		// Railway inyecta PORT; nuestro tag espera SERVER_PORT.
		if os.Getenv("SERVER_PORT") == "" {
			if p := os.Getenv("PORT"); p != "" {
				_ = os.Setenv("SERVER_PORT", p)
			}
		}

		etlCfg = new(ETLConfig)
		if err := gosuite.GetConfigFromEnv(etlCfg); err != nil {
			log.Panicf("error parsing ETL environment vars \n%v", err)
		}
	})
}
