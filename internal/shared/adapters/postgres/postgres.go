package postgres

import (
	"database/sql"
	"fmt"
	"log"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
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

func NewPostgresConnection(config config.Config) *bun.DB {

	configPostgres := config.Postgres
	dns := pgOptions{
		host:     configPostgres.Host,
		port:     configPostgres.Port,
		user:     configPostgres.User,
		password: configPostgres.Password,
		database: configPostgres.Database,
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dns.getDNS())))

	if err := sqldb.Ping(); err != nil {
		log.Fatalf("Error al hacer ping a la base de datos: %v", err)
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
	))

	return db

}
