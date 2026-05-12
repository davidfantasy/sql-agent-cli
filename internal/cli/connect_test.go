package cli

import (
	"bytes"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
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
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "postgres", "--database", "app", "--host", "127.0.0.1", "--port", "5432", "--username", "agent", "--credential-helper", inlineCredentialHelperCommand("secret")})

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

func TestConnectCommand_SavesReadOnlyConnection(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"readonly-test", "--driver", "postgres", "--database", "app", "--credential-helper", inlineCredentialHelperCommand("secret"), "--read-only"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	conn, err := config.LoadConnection("readonly-test")
	if err != nil {
		t.Fatal(err)
	}
	if !conn.ReadOnly {
		t.Fatalf("expected connection to be read-only, got ReadOnly=%v", conn.ReadOnly)
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

func TestConnectCommand_RequiresCredentialSourceOutsideWizard(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "postgres", "--database", "app"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing credential source to fail")
	}
	if !strings.Contains(err.Error(), "--password-env or --credential-helper") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnectCommand_DefaultsPostgresPort(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DB_PASSWORD", "secret")

	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()

	var validated config.Connection
	openConnectionForValidation = func(conn config.Connection) error {
		validated = conn
		return nil
	}

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "postgres", "--database", "app", "--password-env", "DB_PASSWORD"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if validated.Port != 5432 {
		t.Fatalf("expected postgres default port 5432, got %d", validated.Port)
	}

	conn, err := config.LoadConnection("prod")
	if err != nil {
		t.Fatal(err)
	}
	if conn.Port != 5432 {
		t.Fatalf("expected saved postgres port 5432, got %d", conn.Port)
	}
}

func TestConnectCommand_DefaultsMySQLPort(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DB_PASSWORD", "secret")

	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()

	var validated config.Connection
	openConnectionForValidation = func(conn config.Connection) error {
		validated = conn
		return nil
	}

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "mysql", "--database", "app", "--password-env", "DB_PASSWORD"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if validated.Port != 3306 {
		t.Fatalf("expected mysql default port 3306, got %d", validated.Port)
	}
}

func TestConnectCommand_RequiresConfiguredPasswordEnv(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "postgres", "--database", "app", "--password-env", "DB_PASSWORD"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing password env to fail")
	}
	if !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected env guidance in error, got %v", err)
	}

	if _, loadErr := config.LoadConnection("prod"); !os.IsNotExist(loadErr) {
		t.Fatalf("expected failed validation to avoid saving connection, got %v", loadErr)
	}
}

func TestConnectCommand_DoesNotSaveConnectionWhenValidationFails(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DB_PASSWORD", "secret")

	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error {
		return os.ErrPermission
	}

	cmd := newConnectCommand()
	cmd.SetArgs([]string{"prod", "--driver", "postgres", "--database", "app", "--password-env", "DB_PASSWORD"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if _, loadErr := config.LoadConnection("prod"); !os.IsNotExist(loadErr) {
		t.Fatalf("expected validation failure to avoid saving connection, got %v", loadErr)
	}
}

func TestConnectCommand_WizardStoresInlineCredentialAndVerifies(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	originalOpen := openConnectionForValidation
	originalTTY := openWizardTerminal
	originalPassword := readPasswordInput
	defer func() {
		openConnectionForValidation = originalOpen
		openWizardTerminal = originalTTY
		readPasswordInput = originalPassword
	}()

	var validated config.Connection
	openConnectionForValidation = func(conn config.Connection) error {
		validated = conn
		return nil
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	tty := os.NewFile(uintptr(fds[0]), "tty")
	peer := os.NewFile(uintptr(fds[1]), "peer")
	defer tty.Close()
	defer peer.Close()
	if _, err := peer.Write([]byte("postgres\n127.0.0.1\n\napp\nagent\ny\n")); err != nil {
		t.Fatal(err)
	}

	openWizardTerminal = func() (*os.File, error) {
		return tty, nil
	}
	readPasswordInput = func(fd int) ([]byte, error) {
		return []byte("secret"), nil
	}

	cmd := newConnectCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"prod", "--wizard"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if validated.CredentialHelper != inlineCredentialHelperCommand("secret") {
		t.Fatalf("expected inline helper during validation, got %q", validated.CredentialHelper)
	}
	if validated.Port != 5432 {
		t.Fatalf("expected default postgres port, got %d", validated.Port)
	}
	if !validated.ReadOnly {
		t.Fatal("expected wizard read-only prompt to be applied")
	}

	conn, err := config.LoadConnection("prod")
	if err != nil {
		t.Fatal(err)
	}
	if conn.CredentialHelper != inlineCredentialHelperCommand("secret") {
		t.Fatalf("expected stored inline helper, got %q", conn.CredentialHelper)
	}
	if conn.PasswordEnv != "" {
		t.Fatalf("expected password env to be cleared in wizard mode, got %q", conn.PasswordEnv)
	}
	if conn.Username != "agent" {
		t.Fatalf("expected stored username, got %q", conn.Username)
	}
	if conn.Host != "127.0.0.1" {
		t.Fatalf("expected stored host, got %q", conn.Host)
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
