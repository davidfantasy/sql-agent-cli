package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
	"github.com/davidfantasy/sql-agent-cli/internal/output"
)

func TestQueryCommand_BlocksDeleteWithoutConfirm(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	connectCmd := newConnectionAddCommand()
	connectCmd.SetArgs([]string{"local", "--driver", "postgres", "--database", "app", "--credential-helper", inlineCredentialHelperCommand("secret")})
	if err := connectCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cmd := newQueryCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"local", "DELETE FROM users"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected delete without confirm to fail")
	}
	if !strings.Contains(buf.String(), "destructive_query_requires_confirmation") {
		t.Fatalf("expected structured confirmation error, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "show this warning to the human") {
		t.Fatalf("expected output to instruct human confirmation boundary, got %q", buf.String())
	}
}

func TestEmitJSON_WritesEnvelope(t *testing.T) {
	buf := &bytes.Buffer{}
	err := emitJSON(buf, output.Envelope{OK: true, Data: map[string]any{"x": 1}})
	if err != nil {
		t.Fatal(err)
	}

	var payload output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.OK {
		t.Fatal("expected ok true")
	}
}

func TestQueryCommand_RejectsWriteOnReadOnlyConnection(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	connectCmd := newConnectionAddCommand()
	connectCmd.SetArgs([]string{"readonly-conn", "--driver", "postgres", "--database", "app", "--credential-helper", inlineCredentialHelperCommand("secret"), "--read-only"})
	if err := connectCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cmd := newQueryCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"readonly-conn", "DELETE FROM users"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected write on read-only connection to fail")
	}
	if !strings.Contains(buf.String(), "read_only_connection_rejected") {
		t.Fatalf("expected read_only_connection_rejected error, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "readonly-conn") {
		t.Fatalf("expected error to mention connection name, got %q", buf.String())
	}
}

func TestQueryCommand_ReadOnlyConnectionAllowsRead(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	connectCmd := newConnectionAddCommand()
	connectCmd.SetArgs([]string{"readonly-conn", "--driver", "postgres", "--database", "app", "--credential-helper", inlineCredentialHelperCommand("secret"), "--read-only"})
	if err := connectCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cmd := newQueryCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"readonly-conn", "SELECT 1"})

	_ = cmd.Execute()
	if strings.Contains(buf.String(), "read_only_connection_rejected") {
		t.Fatalf("did not expect read_only_connection_rejected for SELECT, got %q", buf.String())
	}
}
