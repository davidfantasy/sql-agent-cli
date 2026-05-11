package credentials

import (
	"testing"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
)

func TestParseHelperResponse_ValidJSON(t *testing.T) {
	resp, err := ParseHelperResponse([]byte(`{"ok":true,"credentials":{"username":"agent","password":"secret"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Credentials.Username != "agent" {
		t.Fatalf("expected agent, got %q", resp.Credentials.Username)
	}
}

func TestResolve_UsesEnvPasswordAndHelperOverrides(t *testing.T) {
	t.Setenv("DB_PASSWORD", "from-env")

	conn := config.Connection{
		Driver:           "postgres",
		Host:             "db.internal",
		Port:             5432,
		Database:         "app",
		Username:         "local-user",
		PasswordEnv:      "DB_PASSWORD",
		CredentialHelper: "helper --profile prod",
	}

	resolved, err := Resolve(conn, func(config.Connection) (HelperCredentials, error) {
		return HelperCredentials{
			Host:     "prod.internal",
			Username: "agent",
			Password: "from-helper",
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Host != "prod.internal" {
		t.Fatalf("expected helper host override, got %q", resolved.Host)
	}
	if resolved.Username != "agent" {
		t.Fatalf("expected helper username override, got %q", resolved.Username)
	}
	if resolved.Password != "from-helper" {
		t.Fatalf("expected helper password override, got %q", resolved.Password)
	}
}

func TestResolve_RequiresDriverAndDatabase(t *testing.T) {
	_, err := Resolve(config.Connection{}, func(config.Connection) (HelperCredentials, error) {
		return HelperCredentials{}, nil
	})
	if err == nil {
		t.Fatal("expected missing required fields to fail")
	}
}
