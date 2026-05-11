package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/david/sql-agent-cli/internal/config"
	"github.com/david/sql-agent-cli/internal/credentials"
	"github.com/david/sql-agent-cli/internal/db"
	"github.com/david/sql-agent-cli/internal/output"
)

func emitJSON(writer io.Writer, payload output.Envelope) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func supportedDriver(driverName string) bool {
	switch strings.ToLower(driverName) {
	case "mysql", "postgres":
		return true
	default:
		return false
	}
}

func resolveConnection(name string) (credentials.Resolved, error) {
	conn, err := config.LoadConnection(name)
	if err != nil {
		return credentials.Resolved{}, err
	}
	return credentials.Resolve(conn, func(connection config.Connection) (credentials.HelperCredentials, error) {
		return credentials.RunCredentialHelper(connection.Name, connection.Driver, connection.CredentialHelper)
	})
}

func openDriver(resolved credentials.Resolved) (db.Driver, error) {
	switch resolved.Driver {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", resolved.Username, resolved.Password, resolved.Host, resolved.Port, resolved.Database)
		return db.NewMySQLDriver(dsn)
	case "postgres":
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", resolved.Username, resolved.Password, resolved.Host, resolved.Port, resolved.Database)
		return db.NewPostgresDriver(dsn)
	default:
		return nil, fmt.Errorf("unsupported driver %q", resolved.Driver)
	}
}
