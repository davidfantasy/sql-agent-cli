package credentials

import (
	"errors"
	"fmt"
	"os"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
)

// Resolved is the fully materialized connection config used by drivers.
type Resolved struct {
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	ReadOnly bool
}

func Resolve(conn config.Connection, helper func(config.Connection) (HelperCredentials, error)) (Resolved, error) {
	resolved := Resolved{
		Driver:   conn.Driver,
		Host:     conn.Host,
		Port:     conn.Port,
		Database: conn.Database,
		Username: conn.Username,
		ReadOnly: conn.ReadOnly,
	}

	if conn.PasswordEnv != "" {
		resolved.Password = os.Getenv(conn.PasswordEnv)
		if resolved.Password == "" {
			return Resolved{}, fmt.Errorf("password env %q is not set; export it before running sql-agent connect, for example: export %s=your-password", conn.PasswordEnv, conn.PasswordEnv)
		}
	}

	if conn.CredentialHelper != "" {
		credentials, err := helper(conn)
		if err != nil {
			return Resolved{}, err
		}
		if credentials.Host != "" {
			resolved.Host = credentials.Host
		}
		if credentials.Port != 0 {
			resolved.Port = credentials.Port
		}
		if credentials.Database != "" {
			resolved.Database = credentials.Database
		}
		if credentials.Username != "" {
			resolved.Username = credentials.Username
		}
		if credentials.Password != "" {
			resolved.Password = credentials.Password
		}
	}

	if resolved.Driver == "" || resolved.Database == "" {
		return Resolved{}, errors.New("connection is missing required fields")
	}

	return resolved, nil
}
