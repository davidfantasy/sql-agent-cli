package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQLDriver(dsn string) (Driver, error) {
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &sqlDriver{db: database, dialect: "mysql"}, nil
}
