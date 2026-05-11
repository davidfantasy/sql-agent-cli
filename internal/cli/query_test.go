package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/david/sql-agent-cli/internal/output"
)

func TestQueryCommand_BlocksDeleteWithoutConfirm(t *testing.T) {
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
