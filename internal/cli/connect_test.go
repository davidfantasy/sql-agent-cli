package cli

import (
	"os"
	"testing"

	"github.com/david/sql-agent-cli/internal/config"
)

func TestConnectCommand_RequiresName(t *testing.T) {
	cmd := newConnectCommand()
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing name to fail")
	}
}

func TestConnectCommand_SavesConnection(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "postgres", "--database", "app", "--host", "127.0.0.1", "--port", "5432", "--username", "agent"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	conn, err := config.LoadConnection("prod")
	if err != nil {
		t.Fatal(err)
	}
	if conn.Driver != "postgres" || conn.Database != "app" || conn.Host != "127.0.0.1" {
		t.Fatalf("unexpected stored connection: %+v", conn)
	}
}

func TestConnectCommand_RejectsUnsupportedDriver(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"local", "--driver", "sqlite", "--database", "./dev.db"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected unsupported driver to fail")
	}
}

func TestDisconnectCommand_RemovesConnection(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	if err := config.SaveConnection(config.Connection{Name: "local", Driver: "mysql", Database: "app", Host: "127.0.0.1", Port: 3306}); err != nil {
		t.Fatal(err)
	}

	cmd := newDisconnectCommand()
	cmd.SetArgs([]string{"local"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := config.LoadConnection("local"); !os.IsNotExist(err) {
		t.Fatalf("expected connection to be deleted, got %v", err)
	}
}
