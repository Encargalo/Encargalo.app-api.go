package postgres

import (
	"database/sql"
	"embed"
	"log"

	"Encargalo.app-api.go/internal/shared/config"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
)

// OLAPDB envuelve la conexión al warehouse para distinguirla de OLTP en dig.
type OLAPDB struct {
	DB *bun.DB
}

//go:embed migrations_olap/*.sql
var embedOLAPMigrations embed.FS

// NewOLAPConnection abre Postgres OLAP y aplica migraciones analytics.
func NewOLAPConnection(cfg config.ETLConfig) *OLAPDB {
	olap := cfg.OlapPostgres
	db := connect(pgOptions{
		host:     olap.Host,
		port:     olap.Port,
		user:     olap.User,
		password: olap.Password,
		database: olap.Database,
	}, etlServiceName)
	migrateOLAP(db.DB)
	return &OLAPDB{DB: db}
}

func migrateOLAP(instance *sql.DB) {
	goose.SetBaseFS(embedOLAPMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Error setting OLAP dialect: %v", err)
	}

	if err := goose.Up(instance, "migrations_olap"); err != nil {
		log.Fatalf("Failed to apply OLAP migrations: %v", err)
	}
}
