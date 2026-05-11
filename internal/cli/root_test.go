package cli

import "testing"

func TestNewRootCommand_HasUseAndSubcommands(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Use != "sql-agent" {
		t.Fatalf("expected root use sql-agent, got %q", cmd.Use)
	}
	if len(cmd.Commands()) == 0 {
		t.Fatal("expected subcommands to be registered")
	}
}
