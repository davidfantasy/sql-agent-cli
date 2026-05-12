package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
)

func TestSchemaCommand_HasListAndDescribe(t *testing.T) {
	cmd := newSchemaCommand()
	if len(cmd.Commands()) != 2 {
		t.Fatalf("expected 2 schema subcommands, got %d", len(cmd.Commands()))
	}
}

func TestCountCommand_RequiresNameAndTarget(t *testing.T) {
	cmd := newCountCommand()
	cmd.SetArgs([]string{"local"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing target to fail")
	}
}

func TestCountCommand_RejectsWriteQueryTarget(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	connectCmd := newConnectCommand()
	connectCmd.SetArgs([]string{"local", "--driver", "postgres", "--database", "app", "--credential-helper", inlineCredentialHelperCommand("secret")})
	if err := connectCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cmd := newCountCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"local", "DELETE FROM users"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected write query target to fail")
	}
	if !strings.Contains(err.Error(), "count target must be a table name or a read-only SELECT query") {
		t.Fatalf("expected read-only count target error, got %v", err)
	}
}

func TestCountCommand_RejectsMultiStatementTableTarget(t *testing.T) {
	t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())
	originalOpen := openConnectionForValidation
	defer func() { openConnectionForValidation = originalOpen }()
	openConnectionForValidation = func(conn config.Connection) error { return nil }

	connectCmd := newConnectCommand()
	connectCmd.SetArgs([]string{"local", "--driver", "postgres", "--database", "app", "--credential-helper", inlineCredentialHelperCommand("secret")})
	if err := connectCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cmd := newCountCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"local", "users; DROP TABLE users"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected multi-statement table target to fail")
	}
	if !strings.Contains(err.Error(), "count target must be a table name or a read-only SELECT query") {
		t.Fatalf("expected read-only count target error, got %v", err)
	}
}

func TestSchemaListCommand_RequiresName(t *testing.T) {
	cmd := newSchemaCommand()
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected schema list without name to fail")
	}
}

func TestSchemaDescribeCommand_RequiresTable(t *testing.T) {
	cmd := newSchemaCommand()
	cmd.SetArgs([]string{"describe", "prod"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected schema describe without table to fail")
	}
}

func TestEmitJSON_ProducesOutputForCommands(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := emitJSON(buf, structEnvelopeForTest()); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected JSON output")
	}
}
