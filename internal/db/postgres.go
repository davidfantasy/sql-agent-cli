package db

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgresDriver(dsn string) (Driver, error) {
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &sqlDriver{db: database, dialect: "postgres"}, nil
}
