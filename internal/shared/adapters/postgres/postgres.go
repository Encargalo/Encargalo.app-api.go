package postgres

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"Encargalo.app-api.go/internal/shared/config"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	apiServiceName           = "encargalo-api"
	etlServiceName           = "encargalo-etl"
	notificationsServiceName = "encargalo-consumer-notifications"
)

const (
	maxOpenConns    = 30
	maxIdleConns    = 30
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 10 * time.Minute
)

type pgOptions struct {
	host     string
	port     int
	user     string
	password string
	database string
}

func (p *pgOptions) getDNS(serviceName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&application_name=%s", p.user, p.password, p.host, p.port, p.database, serviceName)
}

func connect(opts pgOptions, serviceName string) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(opts.getDNS(serviceName))))
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxIdleConns)
	sqldb.SetConnMaxLifetime(connMaxLifetime)
	sqldb.SetConnMaxIdleTime(connMaxIdleTime)

	if err := sqldb.Ping(); err != nil {
		log.Fatalf("Error al hacer ping a la base de datos: %v", err)
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	db.WithQueryHook(queryLogHook{})
	return db
}

// NewPostgresConnection abre OLTP y aplica migraciones goose (API).
func NewPostgresConnection(config config.Config) *bun.DB {
	configPostgres := config.Postgres
	db := connect(pgOptions{
		host:     configPostgres.Host,
		port:     configPostgres.Port,
		user:     configPostgres.User,
		password: configPostgres.Password,
		database: configPostgres.Database,
	}, apiServiceName)
	migrate(db.DB)
	return db
}

// NewOLTPConnection abre OLTP sin correr migraciones (ETL: solo lectura).
func NewOLTPConnection(cfg config.ETLConfig) *bun.DB {
	p := cfg.Postgres
	return connect(pgOptions{
		host:     p.Host,
		port:     p.Port,
		user:     p.User,
		password: p.Password,
		database: p.Database,
	}, etlServiceName)
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func migrate(instance *sql.DB) {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Error setting dialect: %v", err)
	}

	if err := goose.Up(instance, "migrations"); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
}

// NewNotificationsConnection abre OLTP sin migraciones (consumer: solo lectura).
func NewNotificationsConnection(cfg config.NotificationsConfig) *bun.DB {
	p := cfg.Postgres
	return connect(pgOptions{
		host:     p.Host,
		port:     p.Port,
		user:     p.User,
		password: p.Password,
		database: p.Database,
	}, notificationsServiceName)
}
