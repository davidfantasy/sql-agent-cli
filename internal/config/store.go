package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var (
	storeDir = func() string {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".sql-agent", "connections")
	}
)

// Connection is the persisted configuration for a named database target.
type Connection struct {
	Name             string `json:"name"`
	Driver           string `json:"driver"`
	Host             string `json:"host,omitempty"`
	Port             int    `json:"port,omitempty"`
	Database         string `json:"database"`
	Username         string `json:"username,omitempty"`
	PasswordEnv      string `json:"password_env,omitempty"`
	CredentialHelper string `json:"credential_helper,omitempty"`
	ReadOnly         bool   `json:"read_only,omitempty"`
}

// ConnectionSummary is a safe subset of connection info for listing (no secrets).
type ConnectionSummary struct {
	Name     string `json:"name"`
	Driver   string `json:"driver"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database"`
	Username string `json:"username,omitempty"`
	ReadOnly bool   `json:"read_only"`
}

func storePath(name string) string {
	return filepath.Join(storeDir(), name+".enc")
}

func SaveConnection(connection Connection) error {
	data, err := json.Marshal(connection)
	if err != nil {
		return err
	}

	encrypted, err := Encrypt(data)
	if err != nil {
		return err
	}

	path := storePath(connection.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, encrypted, 0o600)
}

func LoadConnection(name string) (Connection, error) {
	data, err := os.ReadFile(storePath(name))
	if err != nil {
		return Connection{}, err
	}

	plaintext, err := Decrypt(data)
	if err != nil {
		return Connection{}, err
	}

	var connection Connection
	if err := json.Unmarshal(plaintext, &connection); err != nil {
		return Connection{}, err
	}

	return connection, nil
}

func DeleteConnection(name string) error {
	err := os.Remove(storePath(name))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListConnections returns a summary of every saved connection.
func ListConnections() ([]ConnectionSummary, error) {
	entries, err := os.ReadDir(storeDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []ConnectionSummary{}, nil
		}
		return nil, err
	}

	var summaries []ConnectionSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".enc") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".enc")
		conn, err := LoadConnection(name)
		if err != nil {
			continue
		}
		summaries = append(summaries, ConnectionSummary{
			Name:     conn.Name,
			Driver:   conn.Driver,
			Host:     conn.Host,
			Port:     conn.Port,
			Database: conn.Database,
			Username: conn.Username,
			ReadOnly: conn.ReadOnly,
		})
	}
	return summaries, nil
}
