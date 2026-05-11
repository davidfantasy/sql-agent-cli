package cli

import (
	"bytes"
	"testing"
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
