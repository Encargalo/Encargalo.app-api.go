package postgres

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"Encargalo.app-api.go/internal/shared/config"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type pgOptions struct {
	host     string
	port     int
	user     string
	password string
	database string
}

func (p *pgOptions) getDNS() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", p.user, p.password, p.host, p.port, p.database)
}

func connect(opts pgOptions) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(opts.getDNS())))

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
	})
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
	})
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
